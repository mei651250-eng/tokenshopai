#!/bin/bash
# ============================================================
# TokenHub 邮件服务配置
# 支持: SMTP (自有/阿里云/腾讯云) / SendGrid / AWS SES
# 使用: sudo bash deploy-email.sh [provider]
# provider: smtp | sendgrid | ses
# 默认: smtp
# ============================================================
set -e

PROVIDER="${1:-smtp}"
INSTALL_DIR="/opt/tokenhub"
CONFIG_FILE="${INSTALL_DIR}/configs/config.yaml"

echo "=========================================="
echo "  TokenHub 邮件服务配置"
echo "  方式: ${PROVIDER}"
echo "=========================================="

# 备份配置
cp "$CONFIG_FILE" "${CONFIG_FILE}.bak.$(date +%s)"

case $PROVIDER in
    smtp)
        echo ""
        echo "--- SMTP 邮件服务配置 ---"
        echo ""
        echo "常见 SMTP 配置:"
        echo "  阿里云企业邮箱: smtp.qiye.aliyun.com:465"
        echo "  腾讯企业邮箱:   smtp.exmail.qq.com:465"
        echo "  QQ 邮箱:        smtp.qq.com:465"
        echo "  163 邮箱:       smtp.163.com:465"
        echo "  Gmail:          smtp.gmail.com:587"
        echo "  Outlook:        smtp.office365.com:587"
        echo ""

        read -p "SMTP 服务器地址: " SMTP_HOST
        read -p "SMTP 端口 [465]: " SMTP_PORT
        SMTP_PORT=${SMTP_PORT:-465}
        read -p "SMTP 用户名 (邮箱): " SMTP_USER
        read -p "SMTP 密码/授权码: " SMTP_PASS
        read -p "发件人名称 [TokenHub]: " FROM_NAME
        FROM_NAME=${FROM_NAME:-TokenHub}
        read -p "发件人邮箱 [${SMTP_USER}]: " FROM_ADDR
        FROM_ADDR=${FROM_ADDR:-$SMTP_USER}

        # 使用 yq 或 sed 更新配置
        if command -v yq &> /dev/null; then
            yq -i ".verification.email.provider = \"smtp\"" "$CONFIG_FILE"
            yq -i ".verification.email.smtp.host = \"${SMTP_HOST}\"" "$CONFIG_FILE"
            yq -i ".verification.email.smtp.port = ${SMTP_PORT}" "$CONFIG_FILE"
            yq -i ".verification.email.smtp.username = \"${SMTP_USER}\"" "$CONFIG_FILE"
            yq -i ".verification.email.smtp.password = \"${SMTP_PASS}\"" "$CONFIG_FILE"
            yq -i ".verification.email.smtp.from_address = \"${FROM_ADDR}\"" "$CONFIG_FILE"
            yq -i ".verification.email.smtp.from_name = \"${FROM_NAME}\"" "$CONFIG_FILE"
        else
            # 使用 sed 替换
            sed -i "s|host: \"smtp.tokenhub.com\"|host: \"${SMTP_HOST}\"|" "$CONFIG_FILE"
            sed -i "s|port: 465|port: ${SMTP_PORT}|" "$CONFIG_FILE"
            sed -i "s|from_address: \"noreply@tokenhub.com\"|from_address: \"${FROM_ADDR}\"|" "$CONFIG_FILE"
            sed -i "s|from_name: \"TokenHub\"|from_name: \"${FROM_NAME}\"|" "$CONFIG_FILE"
            echo ""
            echo "⚠️  未安装 yq，请手动编辑配置文件设置用户名和密码:"
            echo "   vi ${CONFIG_FILE}"
            echo ""
            echo "   需修改的字段:"
            echo "   verification.email.smtp.username: \"${SMTP_USER}\""
            echo "   verification.email.smtp.password: \"你的密码\""
        fi

        echo ""
        echo "SMTP 环境变量（可选，优先级高于配置文件）:"
        cat >> /etc/tokenhub.env <<EOF

# SMTP 邮件配置
SMTP_HOST=${SMTP_HOST}
SMTP_PORT=${SMTP_PORT}
SMTP_USER=${SMTP_USER}
SMTP_PASS=${SMTP_PASS}
SMTP_FROM=${FROM_ADDR}
EOF
        ;;

    sendgrid)
        echo ""
        echo "--- SendGrid 邮件服务配置 ---"
        read -p "SendGrid API Key: " SG_API_KEY
        read -p "发件人邮箱: " FROM_ADDR
        read -p "发件人名称 [TokenHub]: " FROM_NAME
        FROM_NAME=${FROM_NAME:-TokenHub}

        if command -v yq &> /dev/null; then
            yq -i ".verification.email.provider = \"sendgrid\"" "$CONFIG_FILE"
            yq -i ".verification.email.sendgrid.api_key = \"${SG_API_KEY}\"" "$CONFIG_FILE"
            yq -i ".verification.email.sendgrid.from_address = \"${FROM_ADDR}\"" "$CONFIG_FILE"
            yq -i ".verification.email.sendgrid.from_name = \"${FROM_NAME}\"" "$CONFIG_FILE"
        else
            echo "⚠️  请手动编辑 ${CONFIG_FILE}，设置:"
            echo "   verification.email.provider: sendgrid"
            echo "   verification.email.sendgrid.api_key: ${SG_API_KEY}"
            echo "   verification.email.sendgrid.from_address: ${FROM_ADDR}"
        fi

        cat >> /etc/tokenhub.env <<EOF

# SendGrid 邮件
SENDGRID_API_KEY=${SG_API_KEY}
EOF
        ;;

    ses)
        echo ""
        echo "--- AWS SES 邮件服务配置 ---"
        read -p "AWS Access Key ID: " AWS_KEY
        read -p "AWS Secret Access Key: " AWS_SECRET
        read -p "AWS Region [us-east-1]: " AWS_REGION
        AWS_REGION=${AWS_REGION:-us-east-1}
        read -p "发件人邮箱: " FROM_ADDR

        echo "⚠️  AWS SES 需要在后端代码中集成 SDK"
        echo "   当前版本仅支持 SMTP 和 SendGrid"
        echo "   如需使用 SES，请通过 SES 的 SMTP 接口:"
        echo "   Host: email.${AWS_REGION}.amazonaws.com"
        echo "   Port: 465"
        echo "   Username: ${AWS_KEY}"
        echo "   Password: ${AWS_SECRET}"
        echo ""
        echo "   请使用: sudo bash deploy-email.sh smtp"
        echo "   并输入以上 SMTP 信息"
        exit 0
        ;;

    *)
        echo "❌ 不支持的邮件服务: ${PROVIDER}"
        echo "   支持的方式: smtp, sendgrid, ses"
        exit 1
        ;;
esac

# ===== 验证邮件发送 =====
echo ""
read -p "发送测试邮件？输入收件邮箱（留空跳过）: " TEST_EMAIL

if [ -n "$TEST_EMAIL" ]; then
    echo "发送测试邮件到 ${TEST_EMAIL}..."

    # 重启服务以加载新配置
    systemctl restart tokenhub
    sleep 3

    # 通过 API 发送验证码测试
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -X POST http://localhost:8080/auth/verification/send \
        -H "Content-Type: application/json" \
        -d "{\"type\":\"email\",\"target\":\"${TEST_EMAIL}\",\"purpose\":\"test\"}")

    if [ "$HTTP_CODE" = "200" ]; then
        echo "✅ 测试邮件发送请求成功！请检查收件箱"
    else
        echo "⚠️  发送请求返回 HTTP ${HTTP_CODE}，请检查配置"
        echo "   查看服务日志: journalctl -u tokenhub -n 50"
    fi
else
    # 重启服务以加载新配置
    systemctl restart tokenhub
fi

echo ""
echo "=========================================="
echo "  邮件服务配置完成！"
echo ""
echo "  服务方式: ${PROVIDER}"
echo "  配置文件: ${CONFIG_FILE}"
echo "  环境变量: /etc/tokenhub.env"
echo ""
echo "  测试命令:"
echo "    curl -X POST http://localhost:8080/auth/verification/send \\"
echo "      -H 'Content-Type: application/json' \\"
echo "      -d '{\"type\":\"email\",\"target\":\"你的邮箱\",\"purpose\":\"test\"}'"
echo "=========================================="
