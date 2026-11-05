package mailer

const emailLayoutHeader = `
<!DOCTYPE html>
<html lang="pt-BR">
<body style="margin:0;padding:0;background-color:#FAFAF4;font-family:'Segoe UI',Helvetica,Arial,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="padding:32px 16px;">
    <tr>
      <td align="center">
        <table width="480" cellpadding="0" cellspacing="0" style="background-color:#FFFFFF;border-radius:16px;overflow:hidden;">
          <tr>
            <td style="background-color:#173518;padding:24px 32px;">
              <span style="color:#FFFFFF;font-size:20px;font-weight:700;">Ground Guard</span>
            </td>
          </tr>
          <tr>
            <td style="padding:32px;color:#1A1C19;font-size:15px;line-height:1.6;">
`

const emailLayoutFooter = `
            </td>
          </tr>
          <tr>
            <td style="padding:20px 32px;background-color:#EEEEE9;color:#737970;font-size:12px;">
              Se você não fez essa solicitação, pode ignorar este email com segurança.
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>
`

const verificationEmailTemplate = emailLayoutHeader + `
<h2 style="margin-top:0;color:#173518;">Confirme seu email</h2>
<p>Falta só um passo para confirmar sua conta no Ground Guard.</p>
<p style="text-align:center;margin:32px 0;">
  <a href="{{.Link}}" style="background-color:#173518;color:#FFFFFF;text-decoration:none;padding:14px 28px;border-radius:8px;font-weight:600;display:inline-block;">Confirmar meu email</a>
</p>
<p style="color:#737970;font-size:13px;">Se o botão não funcionar, copie e cole este link no navegador:<br>{{.Link}}</p>
<p style="color:#737970;font-size:13px;">Este link expira em 24 horas.</p>
` + emailLayoutFooter

const passwordResetEmailTemplate = emailLayoutHeader + `
<h2 style="margin-top:0;color:#173518;">Redefinir sua senha</h2>
<p>Recebemos um pedido para redefinir a senha da sua conta Ground Guard.</p>
<p style="text-align:center;margin:32px 0;">
  <a href="{{.Link}}" style="background-color:#173518;color:#FFFFFF;text-decoration:none;padding:14px 28px;border-radius:8px;font-weight:600;display:inline-block;">Redefinir senha</a>
</p>
<p style="color:#737970;font-size:13px;">Se o botão não funcionar, copie e cole este link no navegador:<br>{{.Link}}</p>
<p style="color:#737970;font-size:13px;">Este link expira em 1 hora. Se você não pediu isso, ignore este email — sua senha continua a mesma.</p>
` + emailLayoutFooter
