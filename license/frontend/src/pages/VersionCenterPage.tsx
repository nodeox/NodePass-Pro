import { Button, Form, Input, InputNumber, Modal, Popconfirm, Space, Switch, Table, Tabs, Tag, Upload, message } from 'antd'
import type { UploadFile } from 'antd/es/upload/interface'
import dayjs from 'dayjs'
import { useEffect, useState } from 'react'
import type { ProductRelease, VersionPolicy, VersionSyncConfig } from '../types/api'
import { versionApi } from '../utils/api'
import { extractErrorMessage } from '../utils/request'

type ReleaseFormValues = {
  product: string
  version: string
  channel: string
  is_mandatory: boolean
  release_notes?: string
}

type UploadReleaseFormValues = {
  product: string
  version: string
  channel: string
  is_mandatory: boolean
  is_active: boolean
  release_notes?: string
}

type EditReleaseFormValues = {
  product: string
  version: string
  channel: string
  is_mandatory: boolean
  is_active: boolean
  release_notes?: string
}

type PolicyFormValues = {
  product: string
  channel: string
  min_supported_version: string
  recommended_version?: string
  message?: string
  is_active: boolean
}

type VersionSyncFormValues = {
  enabled: boolean
  auto_sync: boolean
  interval_minutes: number
  github_owner: string
  github_repo: string
  github_token?: string
  channel: string
  include_prerelease: boolean
  api_base_url: string
}

type SyncProduct = 'backend' | 'frontend' | 'nodeclient'

const syncProductOrder: Record<SyncProduct, number> = {
  backend: 1,
  frontend: 2,
  nodeclient: 3
}

const syncProducts: SyncProduct[] = ['backend', 'frontend', 'nodeclient']

function isSyncProduct(value: string): value is SyncProduct {
  return value === 'backend' || value === 'frontend' || value === 'nodeclient'
}

function normalizeSyncProduct(value: string): SyncProduct {
  return isSyncProduct(value) ? value : 'nodeclient'
}

function isFormValidationError(err: unknown): boolean {
  return typeof err === 'object' && err !== null && 'errorFields' in err
}

function formatFileSize(value?: number): string {
  if (!value || value <= 0) return '-'
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  if (value < 1024 * 1024 * 1024) return `${(value / (1024 * 1024)).toFixed(1)} MB`
  return `${(value / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

export default function VersionCenterPage() {
  const [releaseOpen, setReleaseOpen] = useState(false)
  const [uploadOpen, setUploadOpen] = useState(false)
  const [syncOpen, setSyncOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [replaceOpen, setReplaceOpen] = useState(false)
  const [recycleOpen, setRecycleOpen] = useState(false)
  const [policyOpen, setPolicyOpen] = useState(false)

  const [releaseSubmitting, setReleaseSubmitting] = useState(false)
  const [uploadSubmitting, setUploadSubmitting] = useState(false)
  const [syncSubmitting, setSyncSubmitting] = useState(false)
  const [manualSyncing, setManualSyncing] = useState(false)
  const [editSubmitting, setEditSubmitting] = useState(false)
  const [replaceSubmitting, setReplaceSubmitting] = useState(false)
  const [policySubmitting, setPolicySubmitting] = useState(false)

  const [downloadingID, setDownloadingID] = useState<number>()
  const [actionLoadingID, setActionLoadingID] = useState<number>()
  const [restoreLoadingID, setRestoreLoadingID] = useState<number>()
  const [purgeLoadingID, setPurgeLoadingID] = useState<number>()
  const [policyActionLoadingID, setPolicyActionLoadingID] = useState<number>()

  const [editingRelease, setEditingRelease] = useState<ProductRelease | null>(null)
  const [replacingRelease, setReplacingRelease] = useState<ProductRelease | null>(null)
  const [editingPolicy, setEditingPolicy] = useState<VersionPolicy | null>(null)
  const [syncConfigs, setSyncConfigs] = useState<VersionSyncConfig[]>([])
  const [activeSyncProduct, setActiveSyncProduct] = useState<SyncProduct>('backend')
  const [syncConfigLoading, setSyncConfigLoading] = useState(false)

  const [releaseForm] = Form.useForm<ReleaseFormValues>()
  const [uploadForm] = Form.useForm<UploadReleaseFormValues>()
  const [syncForm] = Form.useForm<VersionSyncFormValues>()
  const [editForm] = Form.useForm<EditReleaseFormValues>()
  const [policyForm] = Form.useForm<PolicyFormValues>()

  const [uploadFileList, setUploadFileList] = useState<UploadFile[]>([])
  const [replaceFileList, setReplaceFileList] = useState<UploadFile[]>([])
  const [releases, setReleases] = useState<ProductRelease[]>([])
  const [recycleReleases, setRecycleReleases] = useState<ProductRelease[]>([])
  const [policies, setPolicies] = useState<VersionPolicy[]>([])
  const [loading, setLoading] = useState(false)
  const [recycleLoading, setRecycleLoading] = useState(false)

  const sortSyncConfigs = (items: VersionSyncConfig[]) =>
    [...items].sort((left, right) => {
      const leftKey = normalizeSyncProduct(left.product)
      const rightKey = normalizeSyncProduct(right.product)
      return syncProductOrder[leftKey] - syncProductOrder[rightKey]
    })

  const getSyncConfigByProduct = (product: SyncProduct, items: VersionSyncConfig[] = syncConfigs) =>
    items.find((item) => normalizeSyncProduct(item.product) === product)

  const applySyncFormValues = (config?: VersionSyncConfig) => {
    syncForm.setFieldsValue({
      enabled: config?.enabled ?? false,
      auto_sync: config?.auto_sync ?? false,
      interval_minutes: config?.interval_minutes ?? 60,
      github_owner: config?.github_owner ?? '',
      github_repo: config?.github_repo ?? '',
      github_token: '',
      channel: config?.channel ?? 'stable',
      include_prerelease: config?.include_prerelease ?? false,
      api_base_url: config?.api_base_url ?? 'https://api.github.com'
    })
  }

  const load = async () => {
    setLoading(true)
    try {
      const [releaseList, policyList] = await Promise.all([versionApi.listReleases(), versionApi.listPolicies()])
      setReleases(releaseList)
      setPolicies(policyList)
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const loadRecycle = async () => {
    setRecycleLoading(true)
    try {
      setRecycleReleases(await versionApi.listRecycleReleases())
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setRecycleLoading(false)
    }
  }

  const loadSyncConfigs = async () => {
    setSyncConfigLoading(true)
    try {
      const items = sortSyncConfigs(await versionApi.getSyncConfigs())
      setSyncConfigs(items)

      const currentProduct = items.some((item) => normalizeSyncProduct(item.product) === activeSyncProduct) ? activeSyncProduct : 'backend'
      setActiveSyncProduct(currentProduct)
      applySyncFormValues(getSyncConfigByProduct(currentProduct, items))
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setSyncConfigLoading(false)
    }
  }

  const createRelease = async () => {
    setReleaseSubmitting(true)
    try {
      const values = await releaseForm.validateFields()
      await versionApi.createRelease({
        product: values.product.trim(),
        version: values.version.trim(),
        channel: values.channel.trim(),
        is_mandatory: values.is_mandatory,
        release_notes: values.release_notes?.trim() ?? ''
      })
      message.success('发布记录创建成功')
      setReleaseOpen(false)
      releaseForm.resetFields()
      await load()
    } catch (err) {
      if (!isFormValidationError(err)) {
        message.error(extractErrorMessage(err))
      }
    } finally {
      setReleaseSubmitting(false)
    }
  }

  const uploadRelease = async () => {
    setUploadSubmitting(true)
    try {
      const values = await uploadForm.validateFields()
      const targetFile = uploadFileList[0]?.originFileObj
      if (!targetFile) {
        message.error('请先选择安装包文件')
        return
      }

      const formData = new FormData()
      formData.append('product', values.product.trim())
      formData.append('version', values.version.trim())
      formData.append('channel', values.channel.trim())
      formData.append('is_mandatory', String(values.is_mandatory))
      formData.append('is_active', String(values.is_active))
      formData.append('release_notes', values.release_notes?.trim() ?? '')
      formData.append('file', targetFile)

      await versionApi.uploadRelease(formData)
      message.success('安装包上传成功，发布记录已创建')
      setUploadOpen(false)
      uploadForm.resetFields()
      setUploadFileList([])
      await load()
    } catch (err) {
      if (!isFormValidationError(err)) {
        message.error(extractErrorMessage(err))
      }
    } finally {
      setUploadSubmitting(false)
    }
  }

  const saveSyncConfig = async () => {
    setSyncSubmitting(true)
    try {
      const values = await syncForm.validateFields()
      const payload: Parameters<typeof versionApi.updateSyncConfig>[0] = {
        product: activeSyncProduct,
        enabled: values.enabled,
        auto_sync: values.auto_sync,
        interval_minutes: values.interval_minutes,
        github_owner: values.github_owner.trim(),
        github_repo: values.github_repo.trim(),
        channel: values.channel.trim(),
        include_prerelease: values.include_prerelease,
        api_base_url: values.api_base_url.trim()
      }
      const token = values.github_token?.trim()
      if (token) {
        payload.github_token = token
      }

      const updated = await versionApi.updateSyncConfig(payload)
      setSyncConfigs((prev) => {
        const exists = prev.some((item) => normalizeSyncProduct(item.product) === normalizeSyncProduct(updated.product))
        if (!exists) {
          return sortSyncConfigs([...prev, updated])
        }
        return sortSyncConfigs(
          prev.map((item) => (normalizeSyncProduct(item.product) === normalizeSyncProduct(updated.product) ? updated : item))
        )
      })
      syncForm.setFieldValue('github_token', '')
      message.success('同步配置已保存')
    } catch (err) {
      if (!isFormValidationError(err)) {
        message.error(extractErrorMessage(err))
      }
    } finally {
      setSyncSubmitting(false)
    }
  }

  const manualSyncMirror = async () => {
    setManualSyncing(true)
    try {
      const result = await versionApi.manualSync({ product: activeSyncProduct })
      message.success(`${result.product} 同步完成：新增 ${result.imported_count}，跳过 ${result.skipped_count}`)
      await Promise.all([load(), loadSyncConfigs()])
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setManualSyncing(false)
    }
  }

  const openEditModal = (record: ProductRelease) => {
    setEditingRelease(record)
    editForm.setFieldsValue({
      product: record.product,
      version: record.version,
      channel: record.channel,
      is_mandatory: record.is_mandatory,
      is_active: record.is_active,
      release_notes: record.release_notes
    })
    setEditOpen(true)
  }

  const submitEditRelease = async () => {
    if (!editingRelease) return
    setEditSubmitting(true)
    try {
      const values = await editForm.validateFields()
      await versionApi.updateRelease(editingRelease.id, {
        product: values.product.trim(),
        version: values.version.trim(),
        channel: values.channel.trim(),
        is_mandatory: values.is_mandatory,
        is_active: values.is_active,
        release_notes: values.release_notes?.trim() ?? ''
      })
      message.success('发布记录更新成功')
      setEditOpen(false)
      setEditingRelease(null)
      editForm.resetFields()
      await load()
    } catch (err) {
      if (!isFormValidationError(err)) {
        message.error(extractErrorMessage(err))
      }
    } finally {
      setEditSubmitting(false)
    }
  }

  const toggleReleaseStatus = async (record: ProductRelease) => {
    setActionLoadingID(record.id)
    try {
      await versionApi.updateRelease(record.id, { is_active: !record.is_active })
      message.success(record.is_active ? '发布已下线' : '发布已上线')
      await load()
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setActionLoadingID(undefined)
    }
  }

  const openReplaceModal = (record: ProductRelease) => {
    setReplacingRelease(record)
    setReplaceFileList([])
    setReplaceOpen(true)
  }

  const submitReplaceFile = async () => {
    if (!replacingRelease) return
    setReplaceSubmitting(true)
    try {
      const targetFile = replaceFileList[0]?.originFileObj
      if (!targetFile) {
        message.error('请先选择替换文件')
        return
      }
      const formData = new FormData()
      formData.append('file', targetFile)
      await versionApi.replaceReleaseFile(replacingRelease.id, formData)
      message.success('安装包替换成功')
      setReplaceOpen(false)
      setReplacingRelease(null)
      setReplaceFileList([])
      await load()
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setReplaceSubmitting(false)
    }
  }

  const deleteRelease = async (record: ProductRelease) => {
    setActionLoadingID(record.id)
    try {
      await versionApi.deleteRelease(record.id)
      message.success('发布记录已移入回收站')
      await load()
      if (recycleOpen) {
        await loadRecycle()
      }
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setActionLoadingID(undefined)
    }
  }

  const restoreRelease = async (record: ProductRelease) => {
    setRestoreLoadingID(record.id)
    try {
      await versionApi.restoreRelease(record.id)
      message.success('发布记录已恢复')
      await Promise.all([load(), loadRecycle()])
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setRestoreLoadingID(undefined)
    }
  }

  const purgeRelease = async (record: ProductRelease) => {
    setPurgeLoadingID(record.id)
    try {
      await versionApi.purgeRelease(record.id)
      message.success('发布记录已永久删除')
      await loadRecycle()
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setPurgeLoadingID(undefined)
    }
  }

  const submitPolicy = async () => {
    setPolicySubmitting(true)
    try {
      const values = await policyForm.validateFields()
      const payload = {
        product: values.product.trim(),
        channel: values.channel.trim(),
        min_supported_version: values.min_supported_version.trim(),
        recommended_version: values.recommended_version?.trim() || undefined,
        message: values.message?.trim() || undefined,
        is_active: values.is_active
      }
      if (editingPolicy) {
        await versionApi.updatePolicy(editingPolicy.id, payload)
        message.success('版本策略更新成功')
      } else {
        await versionApi.createPolicy(payload)
        message.success('版本策略创建成功')
      }
      setPolicyOpen(false)
      setEditingPolicy(null)
      policyForm.resetFields()
      await load()
    } catch (err) {
      if (!isFormValidationError(err)) {
        message.error(extractErrorMessage(err))
      }
    } finally {
      setPolicySubmitting(false)
    }
  }

  const openCreatePolicyModal = () => {
    setEditingPolicy(null)
    policyForm.resetFields()
    policyForm.setFieldsValue({ channel: 'stable', is_active: true })
    setPolicyOpen(true)
  }

  const openEditPolicyModal = (record: VersionPolicy) => {
    setEditingPolicy(record)
    policyForm.setFieldsValue({
      product: record.product,
      channel: record.channel,
      min_supported_version: record.min_supported_version,
      recommended_version: record.recommended_version,
      message: record.message,
      is_active: record.is_active
    })
    setPolicyOpen(true)
  }

  const deletePolicy = async (record: VersionPolicy) => {
    setPolicyActionLoadingID(record.id)
    try {
      await versionApi.deletePolicy(record.id)
      message.success('版本策略已删除')
      await load()
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setPolicyActionLoadingID(undefined)
    }
  }

  const downloadReleasePackage = async (record: ProductRelease) => {
    if (!record.file_name) return
    setDownloadingID(record.id)
    try {
      const { blob, filename } = await versionApi.downloadReleaseFile(record.id)
      const objectURL = URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = objectURL
      anchor.download = filename || record.file_name
      document.body.appendChild(anchor)
      anchor.click()
      document.body.removeChild(anchor)
      URL.revokeObjectURL(objectURL)
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setDownloadingID(undefined)
    }
  }

  const switchSyncProduct = (nextProductRaw: string) => {
    const nextProduct = normalizeSyncProduct(nextProductRaw)
    setActiveSyncProduct(nextProduct)
    applySyncFormValues(getSyncConfigByProduct(nextProduct))
  }

  const activeSyncConfig = getSyncConfigByProduct(activeSyncProduct)

  return (
    <>
      <Space style={{ marginBottom: 16 }}>
        <Button type="primary" onClick={() => setReleaseOpen(true)}>
          新建发布
        </Button>
        <Button type="primary" ghost onClick={() => setUploadOpen(true)}>
          手动上传版本
        </Button>
        <Button
          onClick={() => {
            setSyncOpen(true)
            void loadSyncConfigs()
          }}
        >
          GitHub 镜像同步
        </Button>
        <Button
          onClick={() => {
            setRecycleOpen(true)
            void loadRecycle()
          }}
        >
          发布回收站
        </Button>
        <Button onClick={openCreatePolicyModal}>新建策略</Button>
      </Space>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={releases}
        pagination={false}
        style={{ marginBottom: 20 }}
        scroll={{ x: 1600 }}
        columns={[
          { title: '产品', dataIndex: 'product', width: 140 },
          { title: '版本', dataIndex: 'version', width: 120 },
          { title: '渠道', dataIndex: 'channel', width: 110 },
          {
            title: '发布状态',
            width: 110,
            render: (_, record) => (record.is_active ? <Tag color="green">已上线</Tag> : <Tag color="default">已下线</Tag>)
          },
          {
            title: '升级策略',
            width: 120,
            render: (_, record) =>
              record.is_mandatory ? <Tag color="red">强制升级</Tag> : <Tag color="blue">可选升级</Tag>
          },
          {
            title: '安装包',
            width: 280,
            render: (_, record) => {
              if (!record.file_name) {
                return <Tag>未上传</Tag>
              }
              return (
                <Space direction="vertical" size={0}>
                  <span>{record.file_name}</span>
                  <span style={{ color: '#999', fontSize: 12 }}>
                    {formatFileSize(record.file_size)}
                    {record.file_sha256 ? ` · SHA256 ${record.file_sha256.slice(0, 12)}...` : ''}
                  </span>
                </Space>
              )
            }
          },
          {
            title: '发布时间',
            dataIndex: 'published_at',
            width: 180,
            render: (value?: string) => (value ? dayjs(value).format('YYYY-MM-DD HH:mm') : '-')
          },
          {
            title: '说明',
            dataIndex: 'release_notes',
            width: 260,
            render: (value?: string) => value || '-'
          },
          {
            title: '操作',
            width: 430,
            fixed: 'right',
            render: (_, record) => {
              const actionLoading = actionLoadingID === record.id
              return (
                <Space>
                  <Button size="small" onClick={() => openEditModal(record)} loading={actionLoading}>
                    编辑
                  </Button>
                  <Popconfirm
                    title={record.is_active ? '确认下线该发布？' : '确认上线该发布？'}
                    onConfirm={() => toggleReleaseStatus(record)}
                    okText="确认"
                    cancelText="取消"
                  >
                    <Button size="small" loading={actionLoading}>
                      {record.is_active ? '下线' : '上线'}
                    </Button>
                  </Popconfirm>
                  <Button size="small" onClick={() => openReplaceModal(record)} loading={actionLoading}>
                    替换安装包
                  </Button>
                  <Button
                    size="small"
                    onClick={() => void downloadReleasePackage(record)}
                    disabled={!record.file_name}
                    loading={downloadingID === record.id}
                  >
                    下载
                  </Button>
                  <Popconfirm
                    title="确认删除该发布？"
                    description="会移入回收站，可在回收站恢复。"
                    onConfirm={() => deleteRelease(record)}
                    okText="确认"
                    cancelText="取消"
                  >
                    <Button danger size="small" loading={actionLoading}>
                      删除
                    </Button>
                  </Popconfirm>
                </Space>
              )
            }
          }
        ]}
      />

      <Table
        rowKey="id"
        loading={loading}
        dataSource={policies}
        pagination={false}
        scroll={{ x: 1000 }}
        columns={[
          { title: '产品', dataIndex: 'product', width: 160 },
          { title: '渠道', dataIndex: 'channel', width: 120 },
          { title: '最低支持版本', dataIndex: 'min_supported_version', width: 180 },
          { title: '推荐版本', dataIndex: 'recommended_version', width: 160 },
          {
            title: '状态',
            width: 120,
            render: (_, record) => (record.is_active ? <Tag color="green">启用</Tag> : <Tag color="default">停用</Tag>)
          },
          { title: '提示信息', dataIndex: 'message' },
          {
            title: '操作',
            width: 180,
            fixed: 'right',
            render: (_, record) => (
              <Space>
                <Button size="small" onClick={() => openEditPolicyModal(record)} loading={policyActionLoadingID === record.id}>
                  编辑
                </Button>
                <Popconfirm
                  title="确认删除该策略？"
                  description="删除后将不再参与版本校验。"
                  onConfirm={() => deletePolicy(record)}
                  okText="确认"
                  cancelText="取消"
                >
                  <Button danger size="small" loading={policyActionLoadingID === record.id}>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            )
          }
        ]}
      />

      <Modal
        title="新建发布"
        open={releaseOpen}
        onCancel={() => setReleaseOpen(false)}
        onOk={() => void createRelease()}
        confirmLoading={releaseSubmitting}
        destroyOnClose
      >
        <Form form={releaseForm} layout="vertical" initialValues={{ channel: 'stable', is_mandatory: false, release_notes: '' }}>
          <Form.Item label="产品" name="product" rules={[{ required: true, message: '请输入产品' }]}>
            <Input placeholder="backend/nodeclient/frontend" />
          </Form.Item>
          <Form.Item label="版本" name="version" rules={[{ required: true, message: '请输入版本号' }]}>
            <Input placeholder="1.2.3" />
          </Form.Item>
          <Form.Item label="渠道" name="channel" rules={[{ required: true, message: '请输入渠道' }]}>
            <Input placeholder="stable" />
          </Form.Item>
          <Form.Item label="强制升级" name="is_mandatory" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item label="发布说明" name="release_notes">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="手动上传版本"
        open={uploadOpen}
        onCancel={() => {
          setUploadOpen(false)
          uploadForm.resetFields()
          setUploadFileList([])
        }}
        onOk={() => void uploadRelease()}
        confirmLoading={uploadSubmitting}
        destroyOnClose
      >
        <Form
          form={uploadForm}
          layout="vertical"
          initialValues={{ channel: 'stable', is_mandatory: false, is_active: true, release_notes: '' }}
        >
          <Form.Item label="产品" name="product" rules={[{ required: true, message: '请输入产品' }]}>
            <Input placeholder="nodeclient/backend/frontend" />
          </Form.Item>
          <Form.Item label="版本" name="version" rules={[{ required: true, message: '请输入版本号' }]}>
            <Input placeholder="1.2.3" />
          </Form.Item>
          <Form.Item label="渠道" name="channel" rules={[{ required: true, message: '请输入渠道' }]}>
            <Input placeholder="stable" />
          </Form.Item>
          <Form.Item label="强制升级" name="is_mandatory" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item label="启用发布" name="is_active" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item label="发布说明" name="release_notes">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item label="安装包文件" required>
            <Upload.Dragger
              multiple={false}
              maxCount={1}
              fileList={uploadFileList}
              beforeUpload={() => false}
              onChange={({ fileList }) => setUploadFileList(fileList.slice(-1))}
              onRemove={() => {
                setUploadFileList([])
                return true
              }}
            >
              <p style={{ marginBottom: 8 }}>点击或拖拽上传安装包（单文件，最大 512MB）</p>
              <p style={{ color: '#999' }}>支持任意二进制/压缩包格式，例如 .tar.gz / .zip / .exe</p>
            </Upload.Dragger>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="GitHub 镜像同步"
        open={syncOpen}
        width={720}
        onCancel={() => setSyncOpen(false)}
        onOk={() => void saveSyncConfig()}
        confirmLoading={syncSubmitting}
        okText="保存配置"
        destroyOnClose
      >
        <Form
          form={syncForm}
          layout="vertical"
          initialValues={{
            enabled: false,
            auto_sync: false,
            interval_minutes: 60,
            github_owner: '',
            github_repo: '',
            github_token: '',
            channel: 'stable',
            include_prerelease: false,
            api_base_url: 'https://api.github.com'
          }}
        >
          <Tabs
            size="small"
            activeKey={activeSyncProduct}
            onChange={switchSyncProduct}
            items={syncProducts.map((item) => ({ key: item, label: item }))}
            style={{ marginBottom: 12 }}
          />

          <Space style={{ marginBottom: 12 }}>
            <Button type="primary" onClick={() => void manualSyncMirror()} loading={manualSyncing} disabled={syncConfigLoading}>
              立即拉取
            </Button>
            <Button onClick={() => void loadSyncConfigs()} loading={syncConfigLoading}>
              刷新状态
            </Button>
          </Space>

          <Form.Item label="启用镜像同步" name="enabled" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item label="自动拉取" name="auto_sync" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item label="自动拉取间隔（分钟）" name="interval_minutes" rules={[{ required: true, message: '请输入间隔分钟' }]}>
            <InputNumber min={5} max={10080} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item
            label="GitHub Owner"
            name="github_owner"
            dependencies={['enabled']}
            rules={[
              ({ getFieldValue }) => ({
                validator: (_, value) => {
                  if (!getFieldValue('enabled')) {
                    return Promise.resolve()
                  }
                  if (typeof value === 'string' && value.trim() !== '') {
                    return Promise.resolve()
                  }
                  return Promise.reject(new Error('启用同步时必须填写 owner'))
                }
              })
            ]}
          >
            <Input placeholder="如: nodeox" />
          </Form.Item>
          <Form.Item
            label="GitHub Repo"
            name="github_repo"
            dependencies={['enabled']}
            rules={[
              ({ getFieldValue }) => ({
                validator: (_, value) => {
                  if (!getFieldValue('enabled')) {
                    return Promise.resolve()
                  }
                  if (typeof value === 'string' && value.trim() !== '') {
                    return Promise.resolve()
                  }
                  return Promise.reject(new Error('启用同步时必须填写 repo'))
                }
              })
            ]}
          >
            <Input placeholder="如: nodeclient" />
          </Form.Item>
          <Form.Item
            label="GitHub Token（可选）"
            name="github_token"
            extra={activeSyncConfig?.has_github_token ? '已保存令牌，留空则保持不变。' : '可留空。'}
          >
            <Input.Password placeholder="ghp_xxx" />
          </Form.Item>
          <Form.Item label="同步渠道" name="channel" rules={[{ required: true, message: '请输入渠道' }]}>
            <Input placeholder="stable" />
          </Form.Item>
          <Form.Item label="包含预发布版本" name="include_prerelease" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item label="GitHub API Base URL" name="api_base_url" rules={[{ required: true, message: '请输入 API 地址' }]}>
            <Input placeholder="https://api.github.com" />
          </Form.Item>
        </Form>

        <div style={{ marginTop: 8, color: '#666', fontSize: 13 }}>
          <div>当前目标：{activeSyncProduct}</div>
          <div>最近同步状态：{activeSyncConfig?.last_sync_status || '-'}</div>
          <div>最近同步时间：{activeSyncConfig?.last_sync_at ? dayjs(activeSyncConfig.last_sync_at).format('YYYY-MM-DD HH:mm:ss') : '-'}</div>
          <div>最近新增数量：{activeSyncConfig?.last_synced_count ?? 0}</div>
          <div>最近同步信息：{activeSyncConfig?.last_sync_message || '-'}</div>
        </div>
      </Modal>

      <Modal
        title={editingRelease ? `编辑发布 #${editingRelease.id}` : '编辑发布'}
        open={editOpen}
        onCancel={() => {
          setEditOpen(false)
          setEditingRelease(null)
          editForm.resetFields()
        }}
        onOk={() => void submitEditRelease()}
        confirmLoading={editSubmitting}
        destroyOnClose
      >
        <Form form={editForm} layout="vertical" initialValues={{ channel: 'stable', is_mandatory: false, is_active: true }}>
          <Form.Item label="产品" name="product" rules={[{ required: true, message: '请输入产品' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="版本" name="version" rules={[{ required: true, message: '请输入版本号' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="渠道" name="channel" rules={[{ required: true, message: '请输入渠道' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="强制升级" name="is_mandatory" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item label="上线状态" name="is_active" valuePropName="checked">
            <Switch checkedChildren="上线" unCheckedChildren="下线" />
          </Form.Item>
          <Form.Item label="发布说明" name="release_notes">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={replacingRelease ? `替换安装包 #${replacingRelease.id}` : '替换安装包'}
        open={replaceOpen}
        onCancel={() => {
          setReplaceOpen(false)
          setReplacingRelease(null)
          setReplaceFileList([])
        }}
        onOk={() => void submitReplaceFile()}
        confirmLoading={replaceSubmitting}
        destroyOnClose
      >
        <Upload.Dragger
          multiple={false}
          maxCount={1}
          fileList={replaceFileList}
          beforeUpload={() => false}
          onChange={({ fileList }) => setReplaceFileList(fileList.slice(-1))}
          onRemove={() => {
            setReplaceFileList([])
            return true
          }}
        >
          <p style={{ marginBottom: 8 }}>选择新的安装包文件（单文件，最大 512MB）</p>
          <p style={{ color: '#999' }}>提交后将替换当前发布记录的安装包元数据和下载内容</p>
        </Upload.Dragger>
      </Modal>

      <Modal
        title="发布回收站"
        open={recycleOpen}
        width={980}
        onCancel={() => setRecycleOpen(false)}
        footer={null}
        destroyOnClose
      >
        <Table
          rowKey="id"
          loading={recycleLoading}
          dataSource={recycleReleases}
          pagination={false}
          scroll={{ x: 900 }}
          columns={[
            { title: '产品', dataIndex: 'product', width: 140 },
            { title: '版本', dataIndex: 'version', width: 120 },
            { title: '渠道', dataIndex: 'channel', width: 110 },
            {
              title: '安装包',
              width: 260,
              render: (_, record) => record.file_name || '-'
            },
            {
              title: '删除时间',
              dataIndex: 'deleted_at',
              width: 180,
              render: (value?: string | null) => (value ? dayjs(value).format('YYYY-MM-DD HH:mm') : '-')
            },
            {
              title: '操作',
              width: 220,
              fixed: 'right',
              render: (_, record) => (
                <Space>
                  <Popconfirm
                    title="确认恢复该发布？"
                    onConfirm={() => restoreRelease(record)}
                    okText="确认"
                    cancelText="取消"
                  >
                    <Button type="primary" size="small" loading={restoreLoadingID === record.id}>
                      恢复
                    </Button>
                  </Popconfirm>
                  <Popconfirm
                    title="确认永久删除？"
                    description="永久删除后不可恢复，并会删除安装包文件。"
                    onConfirm={() => purgeRelease(record)}
                    okText="确认"
                    cancelText="取消"
                  >
                    <Button danger size="small" loading={purgeLoadingID === record.id}>
                      永久删除
                    </Button>
                  </Popconfirm>
                </Space>
              )
            }
          ]}
        />
      </Modal>

      <Modal
        title={editingPolicy ? `编辑版本策略 #${editingPolicy.id}` : '新建版本策略'}
        open={policyOpen}
        onCancel={() => {
          setPolicyOpen(false)
          setEditingPolicy(null)
          policyForm.resetFields()
        }}
        onOk={() => void submitPolicy()}
        confirmLoading={policySubmitting}
        destroyOnClose
      >
        <Form form={policyForm} layout="vertical" initialValues={{ channel: 'stable', is_active: true }}>
          <Form.Item label="产品" name="product" rules={[{ required: true, message: '请输入产品' }]}>
            <Input placeholder="backend/nodeclient/frontend" />
          </Form.Item>
          <Form.Item label="渠道" name="channel" rules={[{ required: true, message: '请输入渠道' }]}>
            <Input placeholder="stable" />
          </Form.Item>
          <Form.Item label="最低支持版本" name="min_supported_version" rules={[{ required: true, message: '请输入最低支持版本' }]}>
            <Input placeholder="1.0.0" />
          </Form.Item>
          <Form.Item label="推荐版本" name="recommended_version">
            <Input placeholder="1.2.0" />
          </Form.Item>
          <Form.Item label="提示信息" name="message">
            <Input.TextArea rows={3} placeholder="例如：当前版本过低，请尽快升级" />
          </Form.Item>
          <Form.Item label="启用" name="is_active" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}
