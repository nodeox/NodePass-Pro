#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_SCRIPT="${ROOT_DIR}/install.sh"

prompt_yes_no() {
  local prompt="$1"
  local default_value="$2" # y / n
  local input=""
  local show_default="y/N"
  if [[ "$default_value" == "y" ]]; then
    show_default="Y/n"
  fi
  while true; do
    read -r -p "${prompt} (${show_default}): " input
    input="${input:-$default_value}"
    case "${input,,}" in
      y|yes) return 0 ;;
      n|no) return 1 ;;
      *) echo "请输入 y 或 n" ;;
    esac
  done
}

sanitize_domain() {
  local input="$1"
  input="${input#http://}"
  input="${input#https://}"
  input="${input%%/*}"
  echo "$input"
}

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}NodePass-Pro 快速部署引导${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

if [[ ! -f "${INSTALL_SCRIPT}" ]]; then
  echo -e "${YELLOW}错误：未找到 install.sh，请在项目根目录运行此脚本${NC}"
  exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
  echo -e "${YELLOW}错误：缺少 curl，请先安装后重试${NC}"
  exit 1
fi

echo "默认策略："
echo "1) 默认使用预构建镜像部署（不本地构建镜像）"
echo "2) 默认最小部署清单（不保留源码）"
echo "3) 默认不构建 nodeclient 下载包"
echo ""

license_key="${LICENSE_KEY:-${NODEPASS_LICENSE_KEY:-}}"
if [[ -z "${license_key}" ]]; then
  while true; do
    read -r -p "请输入授权码（必填）: " license_key
    if [[ -n "${license_key}" ]]; then
      break
    fi
    echo "授权码不能为空。"
  done
fi

license_domain=""
while true; do
  read -r -p "请输入授权绑定域名（必填）: " license_domain
  license_domain="$(sanitize_domain "${license_domain}")"
  if [[ -n "${license_domain}" ]]; then
    break
  fi
  echo "授权绑定域名不能为空。"
done

read -r -p "授权站点地址（可选，如 https://${license_domain}）: " license_site_url

with_caddy=false
frontend_domain=""
backend_domain=""
caddy_email=""
if prompt_yes_no "是否启用 Caddy 自动 HTTPS 反代" "y"; then
  with_caddy=true
  while true; do
    read -r -p "前端域名（必填）: " frontend_domain
    frontend_domain="$(sanitize_domain "${frontend_domain}")"
    if [[ -n "${frontend_domain}" ]]; then
      break
    fi
    echo "前端域名不能为空。"
  done
  read -r -p "后端域名（可选）: " backend_domain
  backend_domain="$(sanitize_domain "${backend_domain}")"
  read -r -p "证书邮箱（可选）: " caddy_email
fi

with_source=false
build_nodeclient=false
build_image=false

if prompt_yes_no "是否保留源码部署（--with-source）" "n"; then
  with_source=true
  if prompt_yes_no "是否构建 nodeclient 下载包（--build-nodeclient）" "n"; then
    build_nodeclient=true
  fi
fi

if prompt_yes_no "是否本地构建镜像（--build-image）" "n"; then
  build_image=true
fi

args=(--non-interactive --license-key "${license_key}" --license-domain "${license_domain}")
if [[ -n "${license_site_url}" ]]; then
  args+=(--license-site-url "${license_site_url}")
fi
if [[ "${with_caddy}" == true ]]; then
  args+=(--with-caddy --frontend-domain "${frontend_domain}")
  if [[ -n "${backend_domain}" ]]; then
    args+=(--backend-domain "${backend_domain}")
  fi
  if [[ -n "${caddy_email}" ]]; then
    args+=(--email "${caddy_email}")
  fi
fi
if [[ "${with_source}" == true ]]; then
  args+=(--with-source)
fi
if [[ "${build_nodeclient}" == true ]]; then
  args+=(--build-nodeclient)
fi
if [[ "${build_image}" == true ]]; then
  args+=(--build-image)
fi

echo ""
echo -e "${GREEN}将执行命令：${NC}"
printf '%q ' "${INSTALL_SCRIPT}" "${args[@]}"
echo ""
echo ""

if ! prompt_yes_no "确认开始部署" "y"; then
  echo "已取消。"
  exit 0
fi

bash "${INSTALL_SCRIPT}" "${args[@]}"
