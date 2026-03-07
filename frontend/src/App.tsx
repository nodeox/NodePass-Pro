import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import { RouterProvider } from 'react-router-dom'

import router from './router'

dayjs.locale('zh-cn')

const App = () => (
  <ConfigProvider locale={zhCN}>
    <RouterProvider router={router} />
  </ConfigProvider>
)

export default App
