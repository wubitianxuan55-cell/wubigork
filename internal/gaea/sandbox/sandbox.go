// Package sandbox wraps a shell command in an OS-level jail so the model's
// `bash` calls are confined: it may read freely but write only inside the
// workspace (plus temp and toolchain caches) and reach the network only when
// allowed. This is the *enforcement* layer beneath the permission rules
// (*policy*): a permitted command still cannot escape the box.
//
// Only macOS (Seatbelt via sandbox-exec) is implemented; on every other OS, or
// when the OS tooling is missing, Command falls back to running the command
// unwrapped (see Available). Confining the in-process file-writer built-ins is
// handled separately, in package tool/builtin.
package sandbox

// Spec describes how to confine one command. The zero value (Mode == "") does
// not enforce, so an unconfigured caller runs commands unchanged.
type Spec struct {
	// Mode is "enforce" to wrap the command, anything else (incl. "off" and "")
	// to run it unwrapped.
	Mode string
	// WriteRoots are directories the command may write to (the workspace root
	// plus any configured extras). Temp dirs and common toolchain caches are
	// added automatically so builds and package managers keep working.
	WriteRoots []string
	// Network allows network egress from inside the sandbox. Off blocks it so a
	// command cannot exfiltrate or fetch; many dev commands (module/package
	// downloads) need it, so it defaults on at the config layer.
	Network bool
	// FailClosed flips the enforce-but-unavailable fallback: by default a
	// platform that cannot confine (Windows without WSL2, for instance) runs
	// the command unwrapped after a startup warning. FailClosed refuses the
	// command instead — 2026-11-08 众神殿结论第四步（pan_1794098356822）：执行链
	// 隔离失败必须拒绝而非裸跑。Consumers decide where to apply it; the field
	// only carries the policy.
	FailClosed bool
}

// enforce reports whether the spec asks for confinement.
func (s Spec) enforce() bool { return s.Mode == "enforce" }
