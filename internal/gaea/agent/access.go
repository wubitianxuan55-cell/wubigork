package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// checkToolAccess applies the path boundary to any tool that declares how to
// resolve its model-controlled targets. Tools without a resolver (network,
// memory, plugins, bash) retain their existing handling.
func (a *AgentRunner) checkToolAccess(ctx context.Context, t tool.Tool, args json.RawMessage) error {
	if a == nil || a.access == nil {
		return nil
	}
	if wp, ok := t.(tool.WritePathResolver); ok {
		paths, err := wp.WritePaths(args)
		if err != nil {
			return fmt.Errorf("%w: cannot resolve write paths for %s: %v", tool.ErrAccessDenied, t.Name(), err)
		}
		for _, path := range paths {
			if err := a.access.CheckWrite(ctx, path); err != nil {
				return err
			}
		}
	}
	if rp, ok := t.(tool.ReadPathResolver); ok {
		paths, err := rp.ReadPaths(ctx, args)
		if err != nil {
			return fmt.Errorf("%w: cannot resolve read paths for %s: %v", tool.ErrAccessDenied, t.Name(), err)
		}
		for _, path := range paths {
			if err := a.access.CheckRead(ctx, path); err != nil {
				return err
			}
		}
	}
	if rt, ok := t.(tool.ReadTargeter); ok {
		path, ok := rt.ReadTarget(args)
		if !ok {
			return fmt.Errorf("%w: cannot resolve read target for %s", tool.ErrAccessDenied, t.Name())
		}
		if err := a.access.CheckRead(ctx, path); err != nil {
			return err
		}
	}
	return nil
}
