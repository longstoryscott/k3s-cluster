#!/bin/bash

cd "$(dirname "$0")" || exit 1
sed -i '' "s|^VITE_BASE_URL=.*|VITE_BASE_URL='https://ai.longstorymedia.com'|" "$(dirname "$0")/.env"
# npm ci
npm run build
# npm run test
# npm run lint
ssh root@longstorymedia.com "for i in \$(ls /var/www/ai.longstorymedia.com); do rm -rf /var/www/ai.longstorymedia.com/\$i; done" || true
rsync -avzru --delete dist/ root@longstorymedia.com:/var/www/ai.longstorymedia.com || true
ssh root@longstorymedia.com "touch /etc/nginx/sites-available/ai.longstorymedia.com" || true
rsync -avzu --delete nginx.conf root@longstorymedia.com:/etc/nginx/sites-available/ai.longstorymedia.com || true
ssh root@longstorymedia.com "ln -s /etc/nginx/sites-available/ai.longstorymedia.com /etc/nginx/sites-enabled/ai.longstorymedia.com" || true
ssh root@longstorymedia.com "chown -R www-data:www-data /var/www/ai.longstorymedia.com" || true
ssh root@longstorymedia.com "systemctl restart nginx" || true

sed -i '' "s|^VITE_BASE_URL=.*|VITE_BASE_URL='http://192.168.0.71:8083'|" "$(dirname "$0")/.env"
# npm ci

cd || exit 1
