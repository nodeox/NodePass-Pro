import { Button, Form, Input, message } from 'antd'

import PageContainer from '../../components/common/PageContainer'
import { benefitCodeApi } from '../../services/api'
import { usePageTitle } from '../../hooks/usePageTitle'
import { getErrorMessage } from '../../utils/error'

type RedeemForm = {
  code: string
}

const RedeemCode = () => {
  usePageTitle('兑换权益码')

  const handleSubmit = async (values: RedeemForm) => {
    try {
      await benefitCodeApi.redeem(values.code)
      message.success('权益码兑换成功')
    } catch (error) {
      message.error(getErrorMessage(error, '兑换失败'))
    }
  }

  return (
    <PageContainer title="兑换权益码" description="输入权益码后自动升级套餐权益。">
      <Form<RedeemForm> layout="vertical" onFinish={handleSubmit}>
        <Form.Item
          label="权益码"
          name="code"
          rules={[{ required: true, message: '请输入权益码' }]}
        >
          <Input placeholder="NP-XXXX-XXXX-XXXX" />
        </Form.Item>
        <Button type="primary" htmlType="submit">
          立即兑换
        </Button>
      </Form>
    </PageContainer>
  )
}

export default RedeemCode
