package fs

import (
	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/observ"
)

// Delete removes a specific version of a module.
func (v *storageImpl) Delete(ctx observ.ProxyContext, module, version string) error {
	const op errors.Op = "fs.Delete"
	ctx, span := observ.StartSpan(ctx, op.String())
	defer span.End()
	versionedPath := v.versionLocation(module, version)
	exists, err := v.Exists(ctx, module, version)
	if err != nil {
		return errors.E(op, err, errors.M(module), errors.V(version))
	}
	if !exists {
		return errors.E(op, errors.M(module), errors.V(version), errors.KindNotFound)
	}
	return v.filesystem.RemoveAll(versionedPath)
}
