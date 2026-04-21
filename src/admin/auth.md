# Authentification

ArchiveTube supports 3 ways to authenticate: `oidc`, `password` and `none`

## Password

For using password authentication, change the mode to `password` in your config and add/change the `password_hash` value to a bcrypt password. Storing the password in bcrypt ensure that if anyone get read access to the database they cannot easily decrypt the password.

You can generate a bcrypt password online with [bcrypt-generator.com](https://bcrypt-generator.com/) or in a terminal using [htpasswd](https://httpd.apache.org/docs/current/programs/htpasswd.html)

```bash
htpasswd -bnBC 12 "" password | tr -d ":"
```

Replace `password` with your password. If you want extra security, increase the bcrypt cost (here 12) to something like 14 or even 16 but this will lead to slower login time (depend on your hardware)


## **O**pen**ID** **C**onnect

If you want to use OIDC for authentifcation, you need to add these values to your config and set auth mode on `oidc`:

```toml
oidc_issuer = "https://auth.example.com"
oidc_client_id = ""
oidc_client_secret = ""
oidc_redirect_url = "https://yt.example.com/auth/callback"
```

> [!WARNING]
> ArchiveTube does not currently supports PKCE (**P**roof **K**ey for **C**ode **E**xchange). This feature is in Roadmap

## None

This auth mode isn't recommended but can be used if you're using ArchiveTube locally or developping it