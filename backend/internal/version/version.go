package version

import "time"

// Version 表示后端服务版本号，可通过 -ldflags 覆盖。
var Version = "1.0.0"

// BuildTime 构建时间，可通过 -ldflags 覆盖。
var BuildTime *time.Time

// GitCommit Git 提交哈希，可通过 -ldflags 覆盖。
var GitCommit *string

// GitBranch Git 分支，可通过 -ldflags 覆盖。
var GitBranch *string

// GoVersion Go 版本，可通过 -ldflags 覆盖。
var GoVersion *string

// GetVersionInfo 获取完整版本信息。
func GetVersionInfo() map[string]interface{} {
	info := map[string]interface{}{
		"version": Version,
	}

	if BuildTime != nil {
		info["build_time"] = BuildTime.Format(time.RFC3339)
	}

	if GitCommit != nil {
		info["git_commit"] = *GitCommit
	}

	if GitBranch != nil {
		info["git_branch"] = *GitBranch
	}

	if GoVersion != nil {
		info["go_version"] = *GoVersion
	}

	return info
}
