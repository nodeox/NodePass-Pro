interface VersionInfo {
  component: string
  version: string
  git_commit?: string
  git_branch?: string
  build_time?: string
  description?: string
}

/**
 * 上报版本信息到授权中心
 */
export async function reportVersion(
  licenseCenterUrl: string,
  token?: string
): Promise<void> {
  const versionInfo: VersionInfo = {
    component: 'frontend',
    version: import.meta.env.VITE_APP_VERSION || 'dev',
    git_commit: import.meta.env.VITE_GIT_COMMIT,
    git_branch: import.meta.env.VITE_GIT_BRANCH,
    build_time: import.meta.env.VITE_BUILD_TIME,
    description: 'Auto-reported from frontend',
  }

  try {
    const response = await fetch(`${licenseCenterUrl}/api/v1/versions/components`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(token && { Authorization: `Bearer ${token}` }),
      },
      body: JSON.stringify(versionInfo),
    })

    if (!response.ok) {
      console.warn('Failed to report version:', response.status)
    } else {
      console.log('Version reported successfully:', versionInfo.version)
    }
  } catch (error) {
    console.warn('Failed to report version:', error)
  }
}

/**
 * 获取当前版本信息
 */
export function getVersionInfo(): VersionInfo {
  return {
    component: 'frontend',
    version: import.meta.env.VITE_APP_VERSION || 'dev',
    git_commit: import.meta.env.VITE_GIT_COMMIT,
    git_branch: import.meta.env.VITE_GIT_BRANCH,
    build_time: import.meta.env.VITE_BUILD_TIME,
  }
}
