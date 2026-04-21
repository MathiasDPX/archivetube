# Authentification

ArchiveTube supports 3 ways to authenticate: `oidc`, `password` and `none`

# Password

For using password authentication, change the mode to `password` in your config and add/change the `password_hash` value to a bcrypt password. Storing the password in bcrypt ensure that if anyone get read access to the database they cannot easily decrypt the password.

You can generate a bcrypt password online with [bcrypt-generator.com](https://bcrypt-generator.com/) or in a terminal using [htpasswd](https://httpd.apache.org/docs/current/programs/htpasswd.html)

```
htpasswd -bnBC 12 "" password | tr -d ":"
```

Replace `password` with your password. If you want extra security, increase the bcrypt cost (here 12) to something like 14 or event 16 but this will lead to slower login time (depend on your hardware)


# **O**pen**ID** **C**onnect (OIDC)

```yaml
oidc_issuer = "https://auth.example.com"
oidc_client_id = ""
oidc_client_secret = ""
oidc_redirect_url = "https://yt.example.com/auth/callback"
```