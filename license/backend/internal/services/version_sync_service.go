package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"nodepass-license-unified/internal/models"

	"gorm.io/gorm"
)

const (
	defaultVersionSyncIntervalMinutes = 60
	minVersionSyncIntervalMinutes     = 5
	maxVersionSyncIntervalMinutes     = 10080
	defaultGitHubAPIBaseURL           = "https://api.github.com"
	defaultVersionSyncProduct         = "nodeclient"
)

var (
	errVersionSyncRunning = errors.New("版本同步任务正在运行")

	versionSyncSupportedProducts = []string{"backend", "frontend", "nodeclient"}
	versionSyncProductOrder      = map[string]int{
		"backend":    1,
		"frontend":   2,
		"nodeclient": 3,
	}
)

// UpdateVersionSyncConfigRequest 更新 GitHub 版本镜像同步配置请求。
type UpdateVersionSyncConfigRequest struct {
	Product           *string `json:"product"`
	Enabled           *bool   `json:"enabled"`
	AutoSync          *bool   `json:"auto_sync"`
	IntervalMinutes   *int    `json:"interval_minutes"`
	GitHubOwner       *string `json:"github_owner"`
	GitHubRepo        *string `json:"github_repo"`
	GitHubToken       *string `json:"github_token"`
	Channel           *string `json:"channel"`
	IncludePrerelease *bool   `json:"include_prerelease"`
	APIBaseURL        *string `json:"api_base_url"`
}

// VersionSyncConfigView 版本镜像同步配置输出。
type VersionSyncConfigView struct {
	ID                uint       `json:"id"`
	Product           string     `json:"product"`
	Enabled           bool       `json:"enabled"`
	AutoSync          bool       `json:"auto_sync"`
	IntervalMinutes   int        `json:"interval_minutes"`
	GitHubOwner       string     `json:"github_owner"`
	GitHubRepo        string     `json:"github_repo"`
	HasGitHubToken    bool       `json:"has_github_token"`
	Channel           string     `json:"channel"`
	IncludePrerelease bool       `json:"include_prerelease"`
	APIBaseURL        string     `json:"api_base_url"`
	LastSyncAt        *time.Time `json:"last_sync_at"`
	LastSyncStatus    string     `json:"last_sync_status"`
	LastSyncMessage   string     `json:"last_sync_message"`
	LastSyncedCount   int        `json:"last_synced_count"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// VersionSyncResult 手动/自动同步执行结果。
type VersionSyncResult struct {
	Product       string    `json:"product"`
	FetchedCount  int       `json:"fetched_count"`
	ImportedCount int       `json:"imported_count"`
	SkippedCount  int       `json:"skipped_count"`
	SyncedAt      time.Time `json:"synced_at"`
	Status        string    `json:"status"`
	Message       string    `json:"message"`
}

type gitHubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	Prerelease  bool   `json:"prerelease"`
	Draft       bool   `json:"draft"`
	PublishedAt string `json:"published_at"`
}

// ListVersionSyncConfigs 查询全部 GitHub 版本镜像同步配置。
func (s *UnifiedService) ListVersionSyncConfigs() ([]VersionSyncConfigView, error) {
	cfgs, err := s.loadVersionSyncConfigs()
	if err != nil {
		return nil, err
	}
	items := make([]VersionSyncConfigView, 0, len(cfgs))
	for _, cfg := range cfgs {
		items = append(items, convertVersionSyncConfigView(cfg))
	}
	return items, nil
}

// GetVersionSyncConfig 查询默认（nodeclient）GitHub 版本镜像同步配置。
func (s *UnifiedService) GetVersionSyncConfig() (*VersionSyncConfigView, error) {
	return s.GetVersionSyncConfigByProduct("")
}

// GetVersionSyncConfigByProduct 查询指定产品的 GitHub 版本镜像同步配置。
func (s *UnifiedService) GetVersionSyncConfigByProduct(product string) (*VersionSyncConfigView, error) {
	targetProduct, err := normalizeVersionSyncProductOrDefault(product)
	if err != nil {
		return nil, err
	}
	cfg, err := s.loadVersionSyncConfigByProduct(targetProduct)
	if err != nil {
		return nil, err
	}
	view := convertVersionSyncConfigView(cfg)
	return &view, nil
}

// UpdateVersionSyncConfig 更新 GitHub 版本镜像同步配置。
func (s *UnifiedService) UpdateVersionSyncConfig(req *UpdateVersionSyncConfigRequest) (*VersionSyncConfigView, error) {
	return s.UpdateVersionSyncConfigWithOperator(req, 0)
}

// UpdateVersionSyncConfigWithOperator 更新 GitHub 版本镜像同步配置（含审计日志）。
func (s *UnifiedService) UpdateVersionSyncConfigWithOperator(req *UpdateVersionSyncConfigRequest, operatorID uint) (*VersionSyncConfigView, error) {
	if req == nil {
		return nil, errors.New("参数无效")
	}

	targetProduct, err := normalizeVersionSyncProductOrDefault(pointerStringValue(req.Product))
	if err != nil {
		return nil, err
	}

	var cfg *models.VersionSyncConfig
	err = s.db.Transaction(func(tx *gorm.DB) error {
		current, innerErr := getOrInitVersionSyncConfig(tx, targetProduct)
		if innerErr != nil {
			return innerErr
		}
		cfg = current

		applyVersionSyncConfigUpdates(cfg, req)
		cfg.Product = targetProduct
		if innerErr = validateVersionSyncConfig(cfg); innerErr != nil {
			return innerErr
		}
		if innerErr = tx.Save(cfg).Error; innerErr != nil {
			return innerErr
		}
		return createAdminAuditLog(tx, operatorID, AuditActionVersionSyncConfigUpdate, "version_sync", map[string]interface{}{
			"config_id":          cfg.ID,
			"product":            cfg.Product,
			"enabled":            cfg.Enabled,
			"auto_sync":          cfg.AutoSync,
			"interval_minutes":   cfg.IntervalMinutes,
			"github_owner":       cfg.GitHubOwner,
			"github_repo":        cfg.GitHubRepo,
			"has_github_token":   strings.TrimSpace(cfg.GitHubToken) != "",
			"channel":            cfg.Channel,
			"include_prerelease": cfg.IncludePrerelease,
			"api_base_url":       cfg.APIBaseURL,
			"last_sync_status":   cfg.LastSyncStatus,
			"last_synced_count":  cfg.LastSyncedCount,
		})
	})
	if err != nil {
		return nil, err
	}

	view := convertVersionSyncConfigView(cfg)
	return &view, nil
}

// ManualSyncVersionMirror 手动执行默认（nodeclient）GitHub 镜像拉取。
func (s *UnifiedService) ManualSyncVersionMirror() (*VersionSyncResult, error) {
	return s.ManualSyncVersionMirrorWithOperator("", 0)
}

// ManualSyncVersionMirrorByProduct 手动执行指定产品 GitHub 镜像拉取。
func (s *UnifiedService) ManualSyncVersionMirrorByProduct(product string) (*VersionSyncResult, error) {
	return s.ManualSyncVersionMirrorWithOperator(product, 0)
}

// ManualSyncVersionMirrorWithOperator 手动执行 GitHub 镜像拉取（含审计日志）。
func (s *UnifiedService) ManualSyncVersionMirrorWithOperator(product string, operatorID uint) (*VersionSyncResult, error) {
	targetProduct, err := normalizeVersionSyncProductOrDefault(product)
	if err != nil {
		return nil, err
	}

	cfg, err := s.loadVersionSyncConfigByProduct(targetProduct)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, fmt.Errorf("%s 镜像同步未启用", targetProduct)
	}

	result, err := s.runVersionMirrorSync(cfg)
	if err != nil {
		return nil, err
	}

	if operatorID > 0 {
		_ = s.db.Transaction(func(tx *gorm.DB) error {
			return createAdminAuditLog(tx, operatorID, AuditActionVersionSyncManual, "version_sync", map[string]interface{}{
				"config_id":      cfg.ID,
				"product":        cfg.Product,
				"fetched_count":  result.FetchedCount,
				"imported_count": result.ImportedCount,
				"skipped_count":  result.SkippedCount,
				"status":         result.Status,
			})
		})
	}

	return result, nil
}

// RunAutoVersionMirrorSync 触发一次自动拉取检查（由后台 ticker 周期调用）。
func (s *UnifiedService) RunAutoVersionMirrorSync() error {
	cfgs, err := s.loadVersionSyncConfigs()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	var firstErr error
	for _, cfg := range cfgs {
		if !cfg.Enabled || !cfg.AutoSync {
			continue
		}

		interval := normalizeVersionSyncInterval(cfg.IntervalMinutes)
		if cfg.LastSyncAt != nil {
			nextDue := cfg.LastSyncAt.UTC().Add(time.Duration(interval) * time.Minute)
			if nextDue.After(now) {
				continue
			}
		}

		if _, runErr := s.runVersionMirrorSync(cfg); runErr != nil {
			if errors.Is(runErr, errVersionSyncRunning) {
				continue
			}
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", cfg.Product, runErr)
			}
		}
	}

	return firstErr
}

func (s *UnifiedService) runVersionMirrorSync(cfg *models.VersionSyncConfig) (*VersionSyncResult, error) {
	productKey := normalizeVersionSyncProduct(cfg.Product)
	if productKey == "" {
		productKey = defaultVersionSyncProduct
	}

	s.versionSyncMu.Lock()
	if s.versionSyncRunning == nil {
		s.versionSyncRunning = make(map[string]bool)
	}
	if s.versionSyncRunning[productKey] {
		s.versionSyncMu.Unlock()
		return nil, errVersionSyncRunning
	}
	s.versionSyncRunning[productKey] = true
	s.versionSyncMu.Unlock()

	defer func() {
		s.versionSyncMu.Lock()
		delete(s.versionSyncRunning, productKey)
		s.versionSyncMu.Unlock()
	}()

	if err := validateVersionSyncConfig(cfg); err != nil {
		_ = s.updateVersionSyncStatus(cfg.ID, "failed", err.Error(), 0)
		return nil, err
	}

	releases, err := fetchGitHubReleases(cfg)
	if err != nil {
		_ = s.updateVersionSyncStatus(cfg.ID, "failed", err.Error(), 0)
		return nil, err
	}

	importedCount := 0
	skippedCount := 0

	err = s.db.Transaction(func(tx *gorm.DB) error {
		for _, rel := range releases {
			if rel.Draft {
				skippedCount++
				continue
			}
			if rel.Prerelease && !cfg.IncludePrerelease {
				skippedCount++
				continue
			}

			version := normalizeGitHubVersion(rel.TagName)
			if version == "" {
				version = normalizeGitHubVersion(rel.Name)
			}
			if version == "" {
				skippedCount++
				continue
			}

			var exists int64
			if err = tx.Unscoped().
				Model(&models.ProductRelease{}).
				Where("product = ? AND channel = ? AND version = ?", cfg.Product, cfg.Channel, version).
				Count(&exists).Error; err != nil {
				return err
			}
			if exists > 0 {
				skippedCount++
				continue
			}

			publishedAt := parseGitHubReleaseTime(rel.PublishedAt)
			release := &models.ProductRelease{
				Product:      cfg.Product,
				Version:      version,
				Channel:      cfg.Channel,
				IsMandatory:  false,
				ReleaseNotes: buildMirroredReleaseNotes(rel.Body, rel.HTMLURL),
				PublishedAt:  publishedAt,
				IsActive:     true,
			}
			if err = tx.Create(release).Error; err != nil {
				return err
			}
			importedCount++
		}

		now := time.Now().UTC()
		message := fmt.Sprintf("拉取完成：新增 %d，跳过 %d", importedCount, skippedCount)
		return tx.Model(&models.VersionSyncConfig{}).
			Where("id = ?", cfg.ID).
			Updates(map[string]interface{}{
				"last_sync_at":      &now,
				"last_sync_status":  "success",
				"last_sync_message": truncateSyncMessage(message),
				"last_synced_count": importedCount,
			}).Error
	})
	if err != nil {
		_ = s.updateVersionSyncStatus(cfg.ID, "failed", err.Error(), 0)
		return nil, err
	}

	result := &VersionSyncResult{
		Product:       cfg.Product,
		FetchedCount:  len(releases),
		ImportedCount: importedCount,
		SkippedCount:  skippedCount,
		SyncedAt:      time.Now().UTC(),
		Status:        "success",
		Message:       fmt.Sprintf("拉取完成：新增 %d，跳过 %d", importedCount, skippedCount),
	}
	return result, nil
}

func (s *UnifiedService) loadVersionSyncConfigByProduct(product string) (*models.VersionSyncConfig, error) {
	targetProduct, err := normalizeVersionSyncProductOrDefault(product)
	if err != nil {
		return nil, err
	}

	var cfg *models.VersionSyncConfig
	err = s.db.Transaction(func(tx *gorm.DB) error {
		item, innerErr := getOrInitVersionSyncConfig(tx, targetProduct)
		if innerErr != nil {
			return innerErr
		}
		cfg = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *UnifiedService) loadVersionSyncConfigs() ([]*models.VersionSyncConfig, error) {
	items := make([]*models.VersionSyncConfig, 0)
	err := s.db.Transaction(func(tx *gorm.DB) error {
		cfgs, innerErr := getOrInitVersionSyncConfigs(tx)
		if innerErr != nil {
			return innerErr
		}
		items = cfgs
		return nil
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func getOrInitVersionSyncConfig(tx *gorm.DB, product string) (*models.VersionSyncConfig, error) {
	targetProduct, err := normalizeVersionSyncProductOrDefault(product)
	if err != nil {
		return nil, err
	}

	cfgs, err := getOrInitVersionSyncConfigs(tx)
	if err != nil {
		return nil, err
	}
	for _, cfg := range cfgs {
		if normalizeVersionSyncProduct(cfg.Product) == targetProduct {
			return cfg, nil
		}
	}

	cfg := newDefaultVersionSyncConfig(targetProduct)
	if err = tx.Create(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func getOrInitVersionSyncConfigs(tx *gorm.DB) ([]*models.VersionSyncConfig, error) {
	var records []models.VersionSyncConfig
	if err := tx.Order("id asc").Find(&records).Error; err != nil {
		return nil, err
	}

	byProduct := make(map[string]*models.VersionSyncConfig)
	for i := range records {
		record := records[i]
		product := normalizeVersionSyncProduct(record.Product)
		if product == "" {
			product = defaultVersionSyncProduct
		}
		if !isSupportedVersionSyncProduct(product) {
			continue
		}

		if record.Product != product {
			if err := tx.Model(&models.VersionSyncConfig{}).Where("id = ?", record.ID).Update("product", product).Error; err != nil {
				return nil, err
			}
			record.Product = product
		}

		if _, exists := byProduct[product]; exists {
			continue
		}

		record.IntervalMinutes = normalizeVersionSyncInterval(record.IntervalMinutes)
		if strings.TrimSpace(record.Channel) == "" {
			record.Channel = "stable"
		}
		if strings.TrimSpace(record.APIBaseURL) == "" {
			record.APIBaseURL = defaultGitHubAPIBaseURL
		}
		if strings.TrimSpace(record.LastSyncStatus) == "" {
			record.LastSyncStatus = "idle"
		}
		if strings.TrimSpace(record.LastSyncMessage) == "" {
			record.LastSyncMessage = "未执行"
		}

		copyRecord := record
		byProduct[product] = &copyRecord
	}

	for _, product := range versionSyncSupportedProducts {
		if _, exists := byProduct[product]; exists {
			continue
		}
		cfg := newDefaultVersionSyncConfig(product)
		if err := tx.Create(&cfg).Error; err != nil {
			return nil, err
		}
		copyCfg := cfg
		byProduct[product] = &copyCfg
	}

	items := make([]*models.VersionSyncConfig, 0, len(versionSyncSupportedProducts))
	for _, product := range versionSyncSupportedProducts {
		if cfg, exists := byProduct[product]; exists {
			items = append(items, cfg)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := versionSyncProductOrder[normalizeVersionSyncProduct(items[i].Product)]
		right := versionSyncProductOrder[normalizeVersionSyncProduct(items[j].Product)]
		return left < right
	})
	return items, nil
}

func newDefaultVersionSyncConfig(product string) models.VersionSyncConfig {
	return models.VersionSyncConfig{
		Product:           product,
		Enabled:           false,
		AutoSync:          false,
		IntervalMinutes:   defaultVersionSyncIntervalMinutes,
		Channel:           "stable",
		IncludePrerelease: false,
		APIBaseURL:        defaultGitHubAPIBaseURL,
		LastSyncStatus:    "idle",
		LastSyncMessage:   "未执行",
	}
}

func applyVersionSyncConfigUpdates(cfg *models.VersionSyncConfig, req *UpdateVersionSyncConfigRequest) {
	if req.Enabled != nil {
		cfg.Enabled = *req.Enabled
	}
	if req.AutoSync != nil {
		cfg.AutoSync = *req.AutoSync
	}
	if req.IntervalMinutes != nil {
		cfg.IntervalMinutes = normalizeVersionSyncInterval(*req.IntervalMinutes)
	}
	if req.GitHubOwner != nil {
		cfg.GitHubOwner = strings.TrimSpace(*req.GitHubOwner)
	}
	if req.GitHubRepo != nil {
		cfg.GitHubRepo = strings.TrimSpace(*req.GitHubRepo)
	}
	if req.GitHubToken != nil {
		token := strings.TrimSpace(*req.GitHubToken)
		if token != "" {
			cfg.GitHubToken = token
		}
	}
	if req.Channel != nil {
		cfg.Channel = strings.TrimSpace(*req.Channel)
	}
	if req.IncludePrerelease != nil {
		cfg.IncludePrerelease = *req.IncludePrerelease
	}
	if req.APIBaseURL != nil {
		baseURL := strings.TrimSpace(*req.APIBaseURL)
		if baseURL == "" {
			baseURL = defaultGitHubAPIBaseURL
		}
		cfg.APIBaseURL = baseURL
	}
}

func validateVersionSyncConfig(cfg *models.VersionSyncConfig) error {
	if cfg == nil {
		return errors.New("配置不存在")
	}
	cfg.Product = normalizeVersionSyncProduct(cfg.Product)
	if cfg.Product == "" {
		cfg.Product = defaultVersionSyncProduct
	}
	if !isSupportedVersionSyncProduct(cfg.Product) {
		return fmt.Errorf("product 仅支持: %s", strings.Join(versionSyncSupportedProducts, ","))
	}

	cfg.IntervalMinutes = normalizeVersionSyncInterval(cfg.IntervalMinutes)
	if strings.TrimSpace(cfg.Channel) == "" {
		return errors.New("channel 不能为空")
	}
	if strings.TrimSpace(cfg.APIBaseURL) == "" {
		cfg.APIBaseURL = defaultGitHubAPIBaseURL
	}
	if cfg.Enabled {
		if strings.TrimSpace(cfg.GitHubOwner) == "" {
			return errors.New("github_owner 不能为空")
		}
		if strings.TrimSpace(cfg.GitHubRepo) == "" {
			return errors.New("github_repo 不能为空")
		}
	}
	return nil
}

func convertVersionSyncConfigView(cfg *models.VersionSyncConfig) VersionSyncConfigView {
	return VersionSyncConfigView{
		ID:                cfg.ID,
		Product:           normalizeVersionSyncProductOrFallback(cfg.Product, defaultVersionSyncProduct),
		Enabled:           cfg.Enabled,
		AutoSync:          cfg.AutoSync,
		IntervalMinutes:   normalizeVersionSyncInterval(cfg.IntervalMinutes),
		GitHubOwner:       cfg.GitHubOwner,
		GitHubRepo:        cfg.GitHubRepo,
		HasGitHubToken:    strings.TrimSpace(cfg.GitHubToken) != "",
		Channel:           cfg.Channel,
		IncludePrerelease: cfg.IncludePrerelease,
		APIBaseURL:        cfg.APIBaseURL,
		LastSyncAt:        cfg.LastSyncAt,
		LastSyncStatus:    cfg.LastSyncStatus,
		LastSyncMessage:   cfg.LastSyncMessage,
		LastSyncedCount:   cfg.LastSyncedCount,
		CreatedAt:         cfg.CreatedAt,
		UpdatedAt:         cfg.UpdatedAt,
	}
}

func normalizeVersionSyncInterval(value int) int {
	switch {
	case value <= 0:
		return defaultVersionSyncIntervalMinutes
	case value < minVersionSyncIntervalMinutes:
		return minVersionSyncIntervalMinutes
	case value > maxVersionSyncIntervalMinutes:
		return maxVersionSyncIntervalMinutes
	default:
		return value
	}
}

func normalizeVersionSyncProduct(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func normalizeVersionSyncProductOrFallback(raw, fallback string) string {
	value := normalizeVersionSyncProduct(raw)
	if value == "" {
		return normalizeVersionSyncProduct(fallback)
	}
	return value
}

func normalizeVersionSyncProductOrDefault(raw string) (string, error) {
	value := normalizeVersionSyncProduct(raw)
	if value == "" {
		value = defaultVersionSyncProduct
	}
	if !isSupportedVersionSyncProduct(value) {
		return "", fmt.Errorf("product 仅支持: %s", strings.Join(versionSyncSupportedProducts, ","))
	}
	return value, nil
}

func isSupportedVersionSyncProduct(product string) bool {
	normalized := normalizeVersionSyncProduct(product)
	for _, item := range versionSyncSupportedProducts {
		if normalized == item {
			return true
		}
	}
	return false
}

func pointerStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func normalizeGitHubVersion(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "v")
	value = strings.TrimPrefix(value, "V")
	if len(value) > 64 {
		value = value[:64]
	}
	return strings.TrimSpace(value)
}

func parseGitHubReleaseTime(raw string) *time.Time {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil
	}
	return &parsed
}

func buildMirroredReleaseNotes(body, sourceURL string) string {
	note := strings.TrimSpace(body)
	source := strings.TrimSpace(sourceURL)
	if source == "" {
		return note
	}
	suffix := fmt.Sprintf("[mirror] source: %s", source)
	if note == "" {
		return suffix
	}
	if strings.Contains(note, source) {
		return note
	}
	return note + "\n\n" + suffix
}

func truncateSyncMessage(message string) string {
	trimmed := strings.TrimSpace(message)
	if len(trimmed) <= 255 {
		return trimmed
	}
	return trimmed[:255]
}

func fetchGitHubReleases(cfg *models.VersionSyncConfig) ([]gitHubRelease, error) {
	baseURL := strings.TrimSpace(cfg.APIBaseURL)
	if baseURL == "" {
		baseURL = defaultGitHubAPIBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	endpoint := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=50", baseURL, url.PathEscape(cfg.GitHubOwner), url.PathEscape(cfg.GitHubRepo))

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token := strings.TrimSpace(cfg.GitHubToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("GitHub API 请求失败: %d %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var releases []gitHubRelease
	if err = json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}
	return releases, nil
}

func (s *UnifiedService) updateVersionSyncStatus(configID uint, status, message string, syncedCount int) error {
	if configID == 0 {
		return nil
	}
	now := time.Now().UTC()
	return s.db.Model(&models.VersionSyncConfig{}).
		Where("id = ?", configID).
		Updates(map[string]interface{}{
			"last_sync_at":      &now,
			"last_sync_status":  strings.TrimSpace(status),
			"last_sync_message": truncateSyncMessage(message),
			"last_synced_count": syncedCount,
		}).Error
}
