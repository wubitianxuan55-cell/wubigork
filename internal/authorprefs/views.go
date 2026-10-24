package authorprefs

type View struct {
	Accepted []Preference `json:"accepted"`
	Pending  []Preference `json:"pending"`
	Conflict []Preference `json:"conflict"`
	Revision int64        `json:"revision"`
}

func viewOf(snaps ...Snapshot) View {
	v := View{}
	for _, snap := range snaps {
		if snap.Revision > v.Revision {
			v.Revision = snap.Revision
		}
		for _, p := range snap.Preferences {
			switch p.Status {
			case StatusAccepted:
				v.Accepted = append(v.Accepted, p)
			case StatusPending:
				v.Pending = append(v.Pending, p)
			case StatusConflict:
				v.Conflict = append(v.Conflict, p)
			}
		}
	}
	return v
}
