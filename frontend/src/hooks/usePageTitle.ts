import { useEffect } from 'react'

const APP_NAME = 'NodePass Panel'

export const usePageTitle = (title: string): void => {
  useEffect(() => {
    document.title = title ? `${title} - ${APP_NAME}` : APP_NAME
  }, [title])
}
