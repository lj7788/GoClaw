const nodemailer = require('nodemailer');

// 默认配置 - 使用 126 邮箱
const DEFAULT_SMTP_CONFIG = {
    host: process.env.SMTP_HOST || 'smtp.126.com',
    port: parseInt(process.env.SMTP_PORT || '465'),
    secure: true,
    auth: {
        user: process.env.SMTP_USER || '账号',
        pass: process.env.SMTP_PASS || '授权码'
    }
};

async function sendEmail(to, subject, text, html = null, smtpConfig = null) {
    try {
        const config = smtpConfig || DEFAULT_SMTP_CONFIG;
        
        const transporter = nodemailer.createTransport({
            host: config.host,
            port: config.port,
            secure: config.secure,
            auth: config.auth
        });

        const mailOptions = {
            from: config.auth.user,
            to: to,
            subject: subject,
            text: text
        };

        if (html) {
            mailOptions.html = html;
        }

        const info = await transporter.sendMail(mailOptions);
        const successMsg = `✅ 邮件发送成功！\n📧 收件人: ${to}\n📋 主题: ${subject}\n🆔 消息ID: ${info.messageId}`;
        console.log(successMsg);
        return { success: true, messageId: info.messageId };
    } catch (error) {
        console.error('Error sending email:', error.message);
        return { success: false, error: error.message };
    }
}

if (require.main === module) {
    // console.log('[DEBUG] 脚本开始执行');
    // console.log('[DEBUG] 命令行参数:', JSON.stringify(process.argv.slice(2)));

    const args = process.argv.slice(2);
    let to = '';
    let subject = '';
    let body = '';
    let recipient = '';
    let content = '';

    for (let i = 0; i < args.length; i++) {
        if (args[i] === '--to' && args[i + 1]) {
            to = args[i + 1];
            // console.log('[DEBUG] 解析到 --to:', to);
        }
        if (args[i] === '--subject' && args[i + 1]) {
            subject = args[i + 1];
            // console.log('[DEBUG] 解析到 --subject:', subject);
        }
        if (args[i] === '--body' && args[i + 1]) {
            body = args[i + 1];
            // console.log('[DEBUG] 解析到 --body:', body);
        }
        if (args[i] === '--recipient' && args[i + 1]) {
            recipient = args[i + 1];
            // console.log('[DEBUG] 解析到 --recipient:', recipient);
        }
        if (args[i] === '--content' && args[i + 1]) {
            content = args[i + 1];
            // console.log('[DEBUG] 解析到 --content:', content);
        }
    }

    if (!to && !recipient) {
        // console.log('[DEBUG] 等待从 stdin 读取 JSON 数据...');
        process.stdin.setEncoding('utf8');
        process.stdin.on('readable', () => {
            let chunk;
            while ((chunk = process.stdin.read()) !== null) {
                // console.log('[DEBUG] 从 stdin 读取到数据:', chunk.toString().substring(0, 200));
                try {
                    const data = JSON.parse(chunk);
                    // console.log('[DEBUG] 解析 JSON:', JSON.stringify(data));
                    if (data.to) to = data.to;
                    if (data.recipient) to = data.recipient;
                    if (data.subject) subject = data.subject;
                    if (data.body) body = data.body;
                    if (data.content) body = data.content;
                } catch (e) {
                    console.log('[ERROR] JSON 解析错误:', e.message);
                }
            }
        });
        process.stdin.on('end', () => {
            // console.log('[DEBUG] stdin 结束');
            to = to || recipient;
            body = body || content;
            // console.log('[DEBUG] 最终参数 - to:', to, 'subject:', subject, 'body:', body);
            if (!to || !subject || !body) {
                console.log('[ERROR] 缺少必要参数');
                console.log('Usage: node index.js --to "email@example.com" --subject "Subject" --body "Body"');
                process.exit(1);
            }
            sendEmail(to, subject, body).then(result => {
                // console.log('[DEBUG] 发送结果:', JSON.stringify(result));
                process.exit(result.success ? 0 : 1);
            });
        });
    } else {
        to = to || recipient;
        body = body || content;
        // console.log('[DEBUG] 从命令行获取参数 - to:', to, 'subject:', subject, 'body:', body);
        if (!to || !subject || !body) {
            console.log('[ERROR] 缺少必要参数');
            console.log('Usage: node index.js --to "email@example.com" --subject "Subject" --body "Body"');
            process.exit(1);
        }
        sendEmail(to, subject, body).then(result => {
            // console.log('[DEBUG] 发送结果:', JSON.stringify(result));
            process.exit(result.success ? 0 : 1);
        });
    }
}

module.exports = { sendEmail };
