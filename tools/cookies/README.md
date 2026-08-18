# Provider cookies — normal workflow

Не вытаскивайте cookies вручную из Chrome/Edge/Firefox.

`VConnect Creator` уже использует отдельную Electron session и умеет экспортировать нужные cookies сам.

## 1. Получить `cookies.zip`

Из `vconnect` repo:

```bash
cd creator-app
npm install
npm start
```

В открывшемся Creator:

1. откройте/создайте Telemost tab и авторизуйтесь в Yandex;
2. если нужен VK work transport — откройте VK tab и авторизуйтесь в VK;
3. нажмите **Export Cookies** в верхней панели;
4. сохранится `cookies.zip` с `cookies-yandex.json` и `cookies-vk.json`.

Формат файлов минимальный:

```json
[
  {"name": "Session_id", "value": "..."}
]
```

## 2. Установить в локальное deployment tree

```bash
python3 tools/cookies/install-cookie-bundle.py ~/Downloads/cookies.zip
```

Для обязательного VK:

```bash
python3 tools/cookies/install-cookie-bundle.py ~/Downloads/cookies.zip --require-vk
```

Скрипт валидирует auth cookie и пишет secrets с правами `0600`.

## 3. Отправить на RU edge

Если repo уже клонирован на сервер:

```bash
scp deploy/ru-edge/secrets/cookies-yandex.json root@RU_SERVER:/opt/zorin-fabric/deploy/ru-edge/secrets/
scp deploy/ru-edge/secrets/cookies-vk.json root@RU_SERVER:/opt/zorin-fabric/deploy/ru-edge/secrets/
ssh root@RU_SERVER 'chmod 600 /opt/zorin-fabric/deploy/ru-edge/secrets/cookies-*.json && cd /opt/zorin-fabric/deploy/ru-edge && docker compose up -d'
```

Cookies являются credential'ами аккаунтов. Не кладите ZIP/JSON в Git и не отправляйте их в issue/chat/log.
