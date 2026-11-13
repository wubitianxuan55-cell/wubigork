# gaea uncommitted worktree snapshot

Base commit: 5c83cf026433dc0e65d9496bcec1241a451d503a

This archive preserves the worktree state at packaging time:
- tracked.patch: binary-capable Git patch containing tracked modifications and deletions.
- worktree-files/: every untracked file that existed at packaging time, including the 14 existing release ZIP/TAR artifacts excluded from Git by policy.
- UNCOMMITTED-FILES.tsv: path, byte size and SHA256 for every restored untracked file.
- TRACKED-FILES.txt and WORKTREE-STATUS.txt: Git accounting at packaging time.

Restore tracked changes with: git apply --binary tracked.patch
Then copy worktree-files/ over the repository root.