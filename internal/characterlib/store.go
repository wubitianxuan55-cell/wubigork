package characterlib

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/assistant"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/whisper"
)

// Store 全局角色库存储（characterlib.db）。
type Store struct {
	db      *sql.DB
	dataDir string
}

// NewStore 创建存储（dataDir 下 characterlib.db）。
func NewStore(dataDir string) *Store {
	return &Store{db: GetDatabase(dataDir), dataDir: dataDir}
}

// Close 关闭底层连接。
func (s *Store) Close() error {
	if s == nil || s.dataDir == "" {
		return nil
	}
	return CloseDatabase(s.dataDir)
}

func nowStr() string { return time.Now().Format("2006-01-02 15:04:05") }

// EnsureBuiltins 把内置人格预设种子化进库（幂等：存在则跳过）。
func (s *Store) EnsureBuiltins(presets []whisper.PersonalityPreset) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("角色库未初始化")
	}
	ts := nowStr()
	for _, p := range presets {
		if c, err := s.Get(p.ID); err == nil && c != nil {
			continue // 已种子化
		}
		c := &Character{
			ID:          p.ID,
			Name:        p.Label,
			Kind:        KindBuiltin,
			Gender:      p.Gender,
			Tags:        append([]string(nil), p.Tags...),
			ChatEnabled: true,
			Dims:        p.Dims,
			VoiceGuide:  p.VoiceGuide,
			CreatedAt:   ts,
			UpdatedAt:   ts,
		}
		if err := s.Upsert(c); err != nil {
			return fmt.Errorf("种子化内置角色 %s 失败: %w", p.ID, err)
		}
	}
	return nil
}

// EnsureAssistants 把虚拟助手同步为库内角色（kind=assistant）。
// assistant 记录仍是微信通道配置的唯一事实源；此处只镜像人格字段。
func (s *Store) EnsureAssistants(assistants []assistant.Assistant, presets []whisper.PersonalityPreset) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("角色库未初始化")
	}
	byID := make(map[string]whisper.PersonalityPreset, len(presets))
	for _, p := range presets {
		byID[p.ID] = p
	}
	ts := nowStr()
	for _, ast := range assistants {
		pid := ast.PersonalityID
		if pid == "" {
			pid = ast.ID
		}
		c, err := s.Get(pid)
		if err != nil || c == nil {
			c = &Character{ID: pid, Kind: KindAssistant, CreatedAt: ts}
		}
		base, _ := byID[pid]
		c.ID = pid
		c.Name = ast.Name
		if c.Name == "" {
			c.Name = base.Label
		}
		c.Kind = KindAssistant
		c.Gender = ast.Gender
		if c.Gender == "" {
			c.Gender = base.Gender
		}
		c.Tags = append([]string(nil), ast.Tags...)
		if len(c.Tags) == 0 {
			c.Tags = append([]string(nil), base.Tags...)
		}
		c.PortraitURL = ast.PortraitURL
		c.VoiceGuide = ast.VoiceGuide
		if c.VoiceGuide == "" {
			c.VoiceGuide = base.VoiceGuide
		}
		c.Dims = ast.Dims
		if c.Dims.T == 0 && c.Dims.I == 0 && c.Dims.S == 0 && c.Dims.O == 0 && c.Dims.R == 0 {
			c.Dims = base.Dims
		}
		c.ChatEnabled = ast.Enabled
		c.AssistantID = ast.ID
		if err := s.Upsert(c); err != nil {
			return fmt.Errorf("同步助手角色 %s 失败: %w", pid, err)
		}
	}
	return nil
}

// Upsert 插入或更新统一角色。
func (s *Store) Upsert(c *Character) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("角色库未初始化")
	}
	if c == nil || c.ID == "" || c.Name == "" {
		return fmt.Errorf("角色 ID 与名称不能为空")
	}
	if c.Kind == "" {
		c.Kind = KindCustom
	}
	ts := nowStr()
	if c.CreatedAt == "" {
		c.CreatedAt = ts
	}
	c.UpdatedAt = ts

	tags, _ := json.Marshal(c.Tags)
	samples, _ := json.Marshal(c.DialogueSamples)
	dims, _ := json.Marshal(c.Dims)
	hidden, _ := json.Marshal(c.HiddenPersona)

	_, err := s.db.Exec(`
		INSERT INTO characters (
			id, name, kind, gender, age, tags, portrait_url,
			role_type, personality, background, appearance, figure, motivation, arc, status, notes, dialogue_samples,
			chat_enabled, dims, voice_guide, behavior_rules, emotion_logic, hidden_persona,
			assistant_id, hidden, created_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, kind=excluded.kind, gender=excluded.gender, age=excluded.age,
			tags=excluded.tags, portrait_url=excluded.portrait_url,
			role_type=excluded.role_type, personality=excluded.personality, background=excluded.background,
			appearance=excluded.appearance, figure=excluded.figure, motivation=excluded.motivation,
			arc=excluded.arc, status=excluded.status, notes=excluded.notes, dialogue_samples=excluded.dialogue_samples,
			chat_enabled=excluded.chat_enabled, dims=excluded.dims, voice_guide=excluded.voice_guide,
			behavior_rules=excluded.behavior_rules, emotion_logic=excluded.emotion_logic,
			hidden_persona=excluded.hidden_persona, assistant_id=excluded.assistant_id,
			hidden=excluded.hidden, updated_at=excluded.updated_at`,
		c.ID, c.Name, c.Kind, c.Gender, c.Age, string(tags), c.PortraitURL,
		c.RoleType, c.Personality, c.Background, c.Appearance, c.Figure, c.Motivation, c.Arc, c.Status, c.Notes, string(samples),
		boolInt(c.ChatEnabled), string(dims), c.VoiceGuide, c.BehaviorRules, c.EmotionLogic, string(hidden),
		c.AssistantID, boolInt(c.Hidden), c.CreatedAt, c.UpdatedAt,
	)
	return err
}

// Get 按 ID 读取角色。
func (s *Store) Get(id string) (*Character, error) {
	if s == nil || s.db == nil || id == "" {
		return nil, fmt.Errorf("角色库未初始化或 ID 为空")
	}
	row := s.db.QueryRow(`
		SELECT id, name, kind, gender, age, tags, portrait_url,
			role_type, personality, background, appearance, figure, motivation, arc, status, notes, dialogue_samples,
			chat_enabled, dims, voice_guide, behavior_rules, emotion_logic, hidden_persona,
			assistant_id, hidden, created_at, updated_at
		FROM characters WHERE id = ?`, id)
	c, err := scanCharacter(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// FindByName 按名称查找角色（导入去重用）。
func (s *Store) FindByName(name string) (*Character, error) {
	if s == nil || s.db == nil || name == "" {
		return nil, nil
	}
	row := s.db.QueryRow(`
		SELECT id, name, kind, gender, age, tags, portrait_url,
			role_type, personality, background, appearance, figure, motivation, arc, status, notes, dialogue_samples,
			chat_enabled, dims, voice_guide, behavior_rules, emotion_logic, hidden_persona,
			assistant_id, hidden, created_at, updated_at
		FROM characters WHERE name = ? AND hidden = 0 ORDER BY updated_at DESC LIMIT 1`, name)
	c, err := scanCharacter(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// Delete 删除角色：内置角色软隐藏，其余硬删并级联清理项目关联。
func (s *Store) Delete(id string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("角色库未初始化")
	}
	c, err := s.Get(id)
	if err != nil {
		return err
	}
	if c == nil {
		return nil
	}
	if c.Kind == KindBuiltin {
		_, err = s.db.Exec("UPDATE characters SET hidden=1, updated_at=? WHERE id=?", nowStr(), id)
		return err
	}
	_, err = s.db.Exec("DELETE FROM characters WHERE id=?", id)
	return err
}

// List 分页查询（query 匹配名称/标签；kind 过滤；chatOnly 只取可聊天角色）。
func (s *Store) List(query, kind string, chatOnly bool, limit, offset int) ([]Character, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("角色库未初始化")
	}
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	where := []string{"hidden = 0"}
	args := []interface{}{}
	if kind != "" {
		where = append(where, "kind = ?")
		args = append(args, kind)
	}
	if query != "" {
		where = append(where, "(name LIKE ? OR tags LIKE ? OR personality LIKE ?)")
		q := "%" + query + "%"
		args = append(args, q, q, q)
	}
	if chatOnly {
		where = append(where, "chat_enabled = 1")
	}
	cond := strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM characters WHERE "+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Query(`
		SELECT id, name, kind, gender, age, tags, portrait_url,
			role_type, personality, background, appearance, figure, motivation, arc, status, notes, dialogue_samples,
			chat_enabled, dims, voice_guide, behavior_rules, emotion_logic, hidden_persona,
			assistant_id, hidden, created_at, updated_at
		FROM characters WHERE `+cond+` ORDER BY updated_at DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Character
	for rows.Next() {
		c, err := scanCharacter(rows)
		if err != nil {
			return nil, 0, err
		}
		if c != nil {
			out = append(out, *c)
		}
	}
	return out, total, rows.Err()
}

// ListChatEnabled 返回所有可聊天角色（聊天人格列表用，按更新时间倒序）。
func (s *Store) ListChatEnabled() []Character {
	items, _, err := s.List("", "", true, 500, 0)
	if err != nil {
		return nil
	}
	return items
}

// Associate 把角色加入项目（幂等，role/arcState/status 为项目内覆盖值）。
func (s *Store) Associate(projectID, charID, role, arcState, status string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("角色库未初始化")
	}
	if projectID == "" || charID == "" {
		return fmt.Errorf("项目与角色 ID 不能为空")
	}
	_, err := s.db.Exec(`
		INSERT INTO project_characters (project_id, character_id, role_in_project, arc_state, status, joined_at)
		VALUES (?,?,?,?,?,?)
		ON CONFLICT(project_id, character_id) DO UPDATE SET
			role_in_project=excluded.role_in_project, arc_state=excluded.arc_state, status=excluded.status`,
		projectID, charID, role, arcState, status, nowStr())
	return err
}

// Dissociate 把角色从项目移除（角色本身保留在全局库）。
func (s *Store) Dissociate(projectID, charID string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("角色库未初始化")
	}
	_, err := s.db.Exec("DELETE FROM project_characters WHERE project_id=? AND character_id=?", projectID, charID)
	return err
}

// ListByProject 返回项目已引用的角色。
func (s *Store) ListByProject(projectID string) ([]ProjectCharacter, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("角色库未初始化")
	}
	rows, err := s.db.Query(`
		SELECT project_id, character_id, role_in_project, arc_state, status, joined_at
		FROM project_characters WHERE project_id=? ORDER BY joined_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProjectCharacter
	for rows.Next() {
		var p ProjectCharacter
		if err := rows.Scan(&p.ProjectID, &p.CharacterID, &p.RoleInProject, &p.ArcState, &p.Status, &p.JoinedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ProjectIDsForCharacter 返回引用了该角色的所有项目（角色全局存在、可被多项目引用）。
func (s *Store) ProjectIDsForCharacter(charID string) ([]string, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("角色库未初始化")
	}
	rows, err := s.db.Query("SELECT project_id FROM project_characters WHERE character_id=? ORDER BY joined_at", charID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ImportProjectCharacters 把项目 characters.json 的角色导入全局库并建立关联（幂等）。
// 去重规则：ID 命中 → 更新；否则名称命中 → 合并；否则以原 ID 新建。
func (s *Store) ImportProjectCharacters(projectID string, chars []types.Character) (int, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("角色库未初始化")
	}
	count := 0
	for _, ch := range chars {
		target, err := s.Get(ch.ID)
		if err != nil {
			return count, err
		}
		if target == nil && ch.Name != "" {
			target, err = s.FindByName(ch.Name)
			if err != nil {
				return count, err
			}
		}
		if target == nil {
			target = &Character{ID: ch.ID, Kind: KindCustom}
		} else if target.ID != ch.ID {
			// 按名称合并：仅补全非空字段，避免薄项目记录覆盖全局库里的丰富设定
			if ch.Gender != "" {
				target.Gender = ch.Gender
			}
			if ch.Age != "" {
				target.Age = ch.Age
			}
			if ch.PortraitURL != "" {
				target.PortraitURL = ch.PortraitURL
			}
			if ch.RoleType != "" {
				target.RoleType = ch.RoleType
			}
			if ch.Personality != "" {
				target.Personality = ch.Personality
			}
			if ch.Background != "" {
				target.Background = ch.Background
			}
			if ch.Appearance != "" {
				target.Appearance = ch.Appearance
			}
			if ch.Figure != "" {
				target.Figure = ch.Figure
			}
			if ch.Motivation != "" {
				target.Motivation = ch.Motivation
			}
			if ch.Arc != "" {
				target.Arc = ch.Arc
			}
			if ch.Status != "" {
				target.Status = ch.Status
			}
			if ch.Notes != "" {
				target.Notes = ch.Notes
			}
		}
		if target.ID != ch.ID {
			target.Name = ch.Name
		} else {
			target.Name = ch.Name
			target.Gender = ch.Gender
			target.Age = ch.Age
			target.PortraitURL = ch.PortraitURL
			target.RoleType = ch.RoleType
			target.Personality = ch.Personality
			target.Background = ch.Background
			target.Appearance = ch.Appearance
			target.Figure = ch.Figure
			target.Motivation = ch.Motivation
			target.Arc = ch.Arc
			target.Status = ch.Status
			target.Notes = ch.Notes
		}
		if err := s.Upsert(target); err != nil {
			return count, err
		}
		if err := s.Associate(projectID, target.ID, ch.RoleType, ch.Arc, ch.Status); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// ProjectCharactersForNovel 物化项目引用 → 小说角色列表。
// 全局字段取自角色本身；弧线/状态取项目关联覆盖值（同一角色在不同小说可不同）。
func (s *Store) ProjectCharactersForNovel(projectID string) ([]types.Character, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("角色库未初始化")
	}
	rows, err := s.db.Query(`
		SELECT c.id, c.name, c.kind, c.gender, c.age, c.tags, c.portrait_url,
			c.role_type, c.personality, c.background, c.appearance, c.figure, c.motivation, c.arc, c.status, c.notes, c.dialogue_samples,
			c.chat_enabled, c.dims, c.voice_guide, c.behavior_rules, c.emotion_logic, c.hidden_persona,
			c.assistant_id, c.hidden, c.created_at, c.updated_at,
			pc.role_in_project, pc.arc_state, pc.status
		FROM project_characters pc JOIN characters c ON c.id = pc.character_id
		WHERE pc.project_id = ? AND c.hidden = 0 ORDER BY pc.joined_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []types.Character
	for rows.Next() {
		var c Character
		var tags, samples, dims, hidden string
		var chatEnabled, hiddenFlag int
		var roleInProject, arcState, status string
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Kind, &c.Gender, &c.Age, &tags, &c.PortraitURL,
			&c.RoleType, &c.Personality, &c.Background, &c.Appearance, &c.Figure, &c.Motivation, &c.Arc, &c.Status, &c.Notes, &samples,
			&chatEnabled, &dims, &c.VoiceGuide, &c.BehaviorRules, &c.EmotionLogic, &hidden,
			&c.AssistantID, &hiddenFlag, &c.CreatedAt, &c.UpdatedAt,
			&roleInProject, &arcState, &status,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(tags), &c.Tags)
		_ = json.Unmarshal([]byte(samples), &c.DialogueSamples)
		_ = json.Unmarshal([]byte(dims), &c.Dims)
		if hidden != "" {
			_ = json.Unmarshal([]byte(hidden), &c.HiddenPersona)
		}
		c.ChatEnabled = chatEnabled != 0
		c.Hidden = hiddenFlag != 0
		if arcState != "" {
			c.Arc = arcState
		}
		if status != "" {
			c.Status = status
		}
		out = append(out, c.ToNovelCharacter())
	}
	return out, rows.Err()
}

func scanCharacter(row interface{ Scan(...any) error }) (*Character, error) {
	var c Character
	var tags, samples, dims, hidden string
	var chatEnabled, hiddenFlag int
	err := row.Scan(
		&c.ID, &c.Name, &c.Kind, &c.Gender, &c.Age, &tags, &c.PortraitURL,
		&c.RoleType, &c.Personality, &c.Background, &c.Appearance, &c.Figure, &c.Motivation, &c.Arc, &c.Status, &c.Notes, &samples,
		&chatEnabled, &dims, &c.VoiceGuide, &c.BehaviorRules, &c.EmotionLogic, &hidden,
		&c.AssistantID, &hiddenFlag, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(tags), &c.Tags)
	_ = json.Unmarshal([]byte(samples), &c.DialogueSamples)
	_ = json.Unmarshal([]byte(dims), &c.Dims)
	if hidden != "" {
		_ = json.Unmarshal([]byte(hidden), &c.HiddenPersona)
	}
	c.ChatEnabled = chatEnabled != 0
	c.Hidden = hiddenFlag != 0
	return &c, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
