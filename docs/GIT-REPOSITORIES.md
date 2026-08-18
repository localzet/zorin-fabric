# Git layout

Не публикуйте release ZIP как один монорепозиторий.

## 1. `zorin-fabric`

Содержит orchestration/integration code:

```text
fabric-controller/
deploy/
docs/
integrations/
tools/
README.md
CHANGELOG.md
.gitignore
```

Здесь **нет** копии Xray/Remnawave и нет полного VConnect source.

## 2. `vconnect`

Содержит весь `vconnect-fabric/` из release bundle как корень отдельного repo:

```text
android-app/
creator-app/
headless/
hooks/
relay/
...
.gitignore
```

Это access transport через VK Звонки / Яндекс Телемост.

## 3. Существующие отдельные repos

- `knotroute` — отдельный overlay product;
- Zorin Trust repos — отдельный trust/identity suite;
- Remnawave/Xray — upstream dependencies. В `zorin-fabric` лежат только adapters/compiler/deploy integration.

## Рекомендуемое расположение на dev-машине

```text
D:/DevDrive/localzet/
  zorin-fabric/
  vconnect/
  knotroute/
  zorin-trust-*/
```

На RU server:

```text
/opt/zorin-fabric
/opt/vconnect
```

## Первый push

Для каждого дерева из `git-ready/`:

```bash
git init -b main
git config user.name "Ivan Zorin"
git config user.email "creator@localzet.com"
git add .
git commit -m "Initial release"
git remote add origin <YOUR_GITHUB_REPO_URL>
git push -u origin main
```

Рекомендуемые имена репозиториев:

```text
zorin-fabric
vconnect
knotroute                 # уже существующий
zorin-trust-runtime       #/остальные Trust repos — как сейчас
```

Секреты должны оставаться только на серверах/локальных машинах. `.gitignore` уже исключает `secrets/`, `.env`, provider cookies и master keys.
