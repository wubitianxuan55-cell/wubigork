package cjk

// CJK 部首归一（PDF 文字提取面）：Chrome/Skia 打印 PDF 时 ToUnicode 会把
// 汉字写成部首兼容字符（文→⽂ U+2F42、黄→⻩ U+2EE9）——画面正确，复制/
// 查找/引用/模型读到的文字全错。康熙部首（U+2F00–U+2FD5）按 NFKC 归一；
// 部首补充（U+2E80–U+2EFF）无 NFKC 分解，按 Unicode
// EquivalentUnifiedIdeograph.txt（16.0）逐对等价，只收目标在 BMP 的条目。
// 不做整体 NFKC：它会把全角标点变成半角。
// 移植自 zai-org/ZCode office-runtime/src/domain/cjk.ts（Apache-2.0，逐字移植）。

import "golang.org/x/text/unicode/norm"

// radicalEquiv 部首区字符 → 等价汉字。init 时补齐康熙段（NFKC 逐点归一）。
var radicalEquiv = map[rune]rune{
	'\u2E81': '\u5382', // ⺁ -> 厂
	'\u2E82': '\u4E5B', // ⺂ -> 乛
	'\u2E83': '\u4E5A', // ⺃ -> 乚
	'\u2E84': '\u4E59', // ⺄ -> 乙
	'\u2E85': '\u4EBB', // ⺅ -> 亻
	'\u2E86': '\u5182', // ⺆ -> 冂
	'\u2E88': '\u5200', // ⺈ -> 刀
	'\u2E89': '\u5202', // ⺉ -> 刂
	'\u2E8A': '\u535C', // ⺊ -> 卜
	'\u2E8B': '\u353E', // ⺋ -> 㔾
	'\u2E8C': '\u5C0F', // ⺌ -> 小
	'\u2E8D': '\u5C0F', // ⺍ -> 小
	'\u2E8E': '\u5140', // ⺎ -> 兀
	'\u2E8F': '\u5C23', // ⺏ -> 尣
	'\u2E90': '\u5C22', // ⺐ -> 尢
	'\u2E92': '\u5DF3', // ⺒ -> 巳
	'\u2E93': '\u5E7A', // ⺓ -> 幺
	'\u2E94': '\u5F51', // ⺔ -> 彑
	'\u2E96': '\u5FC4', // ⺖ -> 忄
	'\u2E97': '\u5FC3', // ⺗ -> 心
	'\u2E98': '\u624C', // ⺘ -> 扌
	'\u2E99': '\u6535', // ⺙ -> 攵
	'\u2E9B': '\u65E1', // ⺛ -> 旡
	'\u2E9C': '\u65E5', // ⺜ -> 日
	'\u2E9D': '\u6708', // ⺝ -> 月
	'\u2E9E': '\u6B7A', // ⺞ -> 歺
	'\u2E9F': '\u6BCD', // ⺟ -> 母
	'\u2EA0': '\u6C11', // ⺠ -> 民
	'\u2EA1': '\u6C35', // ⺡ -> 氵
	'\u2EA2': '\u6C3A', // ⺢ -> 氺
	'\u2EA3': '\u706C', // ⺣ -> 灬
	'\u2EA4': '\u722B', // ⺤ -> 爫
	'\u2EA5': '\u722B', // ⺥ -> 爫
	'\u2EA6': '\u4E2C', // ⺦ -> 丬
	'\u2EA7': '\u725B', // ⺧ -> 牛
	'\u2EA8': '\u72AD', // ⺨ -> 犭
	'\u2EA9': '\u738B', // ⺩ -> 王
	'\u2EAB': '\u76EE', // ⺫ -> 目
	'\u2EAC': '\u793A', // ⺬ -> 示
	'\u2EAD': '\u793B', // ⺭ -> 礻
	'\u2EAF': '\u7CF9', // ⺯ -> 糹
	'\u2EB0': '\u7E9F', // ⺰ -> 纟
	'\u2EB1': '\u7F53', // ⺱ -> 罓
	'\u2EB2': '\u7F52', // ⺲ -> 罒
	'\u2EB3': '\u34C1', // ⺳ -> 㓁
	'\u2EB4': '\u5197', // ⺴ -> 冗
	'\u2EB6': '\u7F8A', // ⺶ -> 羊
	'\u2EB9': '\u8002', // ⺹ -> 耂
	'\u2EBA': '\u8080', // ⺺ -> 肀
	'\u2EBB': '\u807F', // ⺻ -> 聿
	'\u2EBC': '\u8089', // ⺼ -> 肉
	'\u2EBE': '\u8279', // ⺾ -> 艹
	'\u2EBF': '\u8279', // ⺿ -> 艹
	'\u2EC0': '\u8279', // ⻀ -> 艹
	'\u2EC1': '\u864E', // ⻁ -> 虎
	'\u2EC2': '\u8864', // ⻂ -> 衤
	'\u2EC3': '\u8980', // ⻃ -> 覀
	'\u2EC4': '\u897F', // ⻄ -> 西
	'\u2EC5': '\u89C1', // ⻅ -> 见
	'\u2EC6': '\u89D2', // ⻆ -> 角
	'\u2EC8': '\u8BA0', // ⻈ -> 讠
	'\u2EC9': '\u8D1D', // ⻉ -> 贝
	'\u2ECB': '\u8F66', // ⻋ -> 车
	'\u2ECC': '\u8FB6', // ⻌ -> 辶
	'\u2ECD': '\u8FB6', // ⻍ -> 辶
	'\u2ECE': '\u8FB6', // ⻎ -> 辶
	'\u2ECF': '\u9091', // ⻏ -> 邑
	'\u2ED0': '\u9485', // ⻐ -> 钅
	'\u2ED1': '\u9577', // ⻑ -> 長
	'\u2ED2': '\u9578', // ⻒ -> 镸
	'\u2ED3': '\u957F', // ⻓ -> 长
	'\u2ED4': '\u95E8', // ⻔ -> 门
	'\u2ED6': '\u961D', // ⻖ -> 阝
	'\u2ED7': '\u96E8', // ⻗ -> 雨
	'\u2ED8': '\u9752', // ⻘ -> 青
	'\u2ED9': '\u97E6', // ⻙ -> 韦
	'\u2EDA': '\u9875', // ⻚ -> 页
	'\u2EDB': '\u98CE', // ⻛ -> 风
	'\u2EDC': '\u98DE', // ⻜ -> 飞
	'\u2EDD': '\u98DF', // ⻝ -> 食
	'\u2EDF': '\u98E0', // ⻟ -> 飠
	'\u2EE0': '\u9963', // ⻠ -> 饣
	'\u2EE2': '\u9A6C', // ⻢ -> 马
	'\u2EE3': '\u9AA8', // ⻣ -> 骨
	'\u2EE4': '\u9B3C', // ⻤ -> 鬼
	'\u2EE5': '\u9C7C', // ⻥ -> 鱼
	'\u2EE6': '\u9E1F', // ⻦ -> 鸟
	'\u2EE7': '\u5364', // ⻧ -> 卤
	'\u2EE8': '\u9EA6', // ⻨ -> 麦
	'\u2EE9': '\u9EC4', // ⻩ -> 黄
	'\u2EEA': '\u9EFE', // ⻪ -> 黾
	'\u2EEB': '\u6589', // ⻫ -> 斉
	'\u2EEC': '\u9F50', // ⻬ -> 齐
	'\u2EED': '\u6B6F', // ⻭ -> 歯
	'\u2EEE': '\u9F7F', // ⻮ -> 齿
	'\u2EEF': '\u7ADC', // ⻯ -> 竜
	'\u2EF0': '\u9F99', // ⻰ -> 龙
	'\u2EF1': '\u9F9C', // ⻱ -> 龜
	'\u2EF2': '\u4E80', // ⻲ -> 亀
	'\u2EF3': '\u9F9F', // ⻳ -> 龟
}

const (
	kangxiFirst = 0x2F00
	kangxiLast  = 0x2FD5
	suppFirst   = 0x2E80
	suppLast    = 0x2EFF
)

func init() {
	for r := rune(kangxiFirst); r <= rune(kangxiLast); r++ {
		if s := norm.NFKC.String(string(r)); len([]rune(s)) == 1 {
			radicalEquiv[r] = []rune(s)[0]
		}
	}
}

// NormalizeCJKCompat 把部首区字符替换为等价汉字，其余文本逐字不变
// （全角标点、异体字等一律不动）。
func NormalizeCJKCompat(text string) string {
	rs := []rune(text)
	changed := false
	for i, r := range rs {
		if (r >= suppFirst && r <= suppLast) || (r >= kangxiFirst && r <= kangxiLast) {
			if eq, ok := radicalEquiv[r]; ok {
				rs[i] = eq
				changed = true
			}
		}
	}
	if !changed {
		return text
	}
	return string(rs)
}
