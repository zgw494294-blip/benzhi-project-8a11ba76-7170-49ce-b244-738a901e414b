package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/benzhi/oral-history-release/internal/domain"
)

func (r *Repository) Recover() error {
	entries, err := os.ReadDir(filepath.Join(r.root, "cases"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		caseID := strings.TrimSuffix(entry.Name(), ".json")
		record, err := r.readRecord(caseID)
		if err != nil {
			return fmt.Errorf("恢复案卷 %s: %w", caseID, err)
		}
		if record.Aggregate.Revision < 1 {
			return fmt.Errorf("恢复案卷 %s: revision 不合法", caseID)
		}
		if err := domain.ValidateAuditChain(record.Audits); err != nil {
			return fmt.Errorf("恢复案卷 %s: %w", caseID, err)
		}
		for _, digest := range record.ObjectDigests {
			if err := r.validateObject(digest); err != nil {
				return fmt.Errorf("恢复案卷 %s: %w", caseID, err)
			}
		}
		if record.Aggregate.FrozenManifest != nil {
			manifest := record.Aggregate.FrozenManifest
			if domain.RecalculateManifestDigest(*manifest) != manifest.ManifestDigest {
				return fmt.Errorf("恢复案卷 %s: 冻结清单摘要不一致", caseID)
			}
		}
	}
	return nil
}
