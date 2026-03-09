package services

import (
	"encoding/json"

	"nodepass-license-unified/internal/models"

	"gorm.io/gorm"
)

const (
	AuditActionLicenseBatchUpdate      = "license_batch_update"
	AuditActionLicenseBatchRevoke      = "license_batch_revoke"
	AuditActionLicenseBatchRestore     = "license_batch_restore"
	AuditActionLicenseClearActivations = "license_clear_activations"
	AuditActionReleaseUpdate           = "release_update"
	AuditActionReleaseReplacePackage   = "release_replace_package"
	AuditActionReleaseDelete           = "release_delete"
	AuditActionReleaseRestore          = "release_restore"
	AuditActionReleasePurge            = "release_purge"
	AuditActionVersionSyncConfigUpdate = "version_sync_config_update"
	AuditActionVersionSyncManual       = "version_sync_manual"
)

func createAdminAuditLog(tx *gorm.DB, adminID uint, action, targetType string, payload map[string]interface{}) error {
	if adminID == 0 {
		return nil
	}

	payloadJSON := "{}"
	if len(payload) > 0 {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		payloadJSON = string(raw)
	}

	return tx.Create(&models.AdminAuditLog{
		AdminID:     adminID,
		Action:      action,
		TargetType:  targetType,
		PayloadJSON: payloadJSON,
	}).Error
}
