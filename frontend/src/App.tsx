import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import { RouterProvider } from 'react-router-dom'

import router from './router'

dayjs.locale('zh-cn')

const App = () => (
  <ConfigProvider
    locale={zhCN}
    theme={{
      token: {
        colorPrimary: '#1677ff',
        borderRadius: 10,
        wireframe: false,
      },
      components: {
        Layout: {
          headerBg: '#ffffff',
          siderBg: '#ffffff',
          bodyBg: '#f5f7fb',
        },
        Menu: {
          itemBorderRadius: 8,
        },
        Card: {
          borderRadiusLG: 12,
        },
      },
    }}
  >
    <RouterProvider router={router} />
  </ConfigProvider>
)

export default App
