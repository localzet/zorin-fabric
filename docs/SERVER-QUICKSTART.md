# Server quickstart

## Что где запускать

### Controller server

Можно разместить на EU/management server. Он не несёт пользовательский трафик.

```text
/opt/zorin-fabric
  deploy/controller
```

### Каждый RU edge

На сервере, где Remnawave Node/Xray уже обслуживает RU→EU маршруты:

```text
/opt/zorin-fabric
/opt/vconnect
```

VConnect Edge использует host networking, чтобы видеть loopback `FABRIC-VCONNECT-*` inbounds основного Xray.

## Клонирование

```bash
cd /opt
git clone <ZORIN_FABRIC_REPO_URL> zorin-fabric
git clone <VCONNECT_REPO_URL> vconnect
```

## Controller

```bash
cd /opt/zorin-fabric
python3 tools/init-fabric-secrets.py --edges ru-1
# вручную: deploy/controller/secrets/remnawave-api-token.txt
# отредактировать deploy/controller/fabric-controller.json
sudo tools/server/install-controller.sh
```

Controller слушает `127.0.0.1:47800`; наружу выставляйте его через ваш HTTPS reverse proxy.

## RU edge

```bash
cd /opt/zorin-fabric/deploy/ru-edge
cp .env.example .env
nano .env
```

Положите secrets в `deploy/ru-edge/secrets/` и затем:

```bash
sudo /opt/zorin-fabric/tools/server/install-ru-edge.sh
```

## Cookies

На рабочем ПК получите `cookies.zip` из VConnect Creator → **Export Cookies**, затем:

```bash
python3 tools/cookies/install-cookie-bundle.py ~/Downloads/cookies.zip
scp deploy/ru-edge/secrets/cookies-*.json root@RU_SERVER:/opt/zorin-fabric/deploy/ru-edge/secrets/
```

На сервере:

```bash
chmod 600 /opt/zorin-fabric/deploy/ru-edge/secrets/cookies-*.json
cd /opt/zorin-fabric/deploy/ru-edge
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
```

## Update

```bash
cd /opt/zorin-fabric && git pull --ff-only
cd /opt/vconnect && git pull --ff-only
sudo /opt/zorin-fabric/tools/server/install-ru-edge.sh
```
