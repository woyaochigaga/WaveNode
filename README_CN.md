cd /Users/chenjiantao/Documents/project/WaveNode/sub2api

rsync -az --delete \
  --exclude='.git/' \
  --exclude='frontend/node_modules/' \
  --exclude='node_modules/' \
  --exclude='backend/vendor/' \
  --exclude='deploy/.env' \
  --exclude='deploy/data/' \
  --exclude='deploy/postgres_data/' \
  --exclude='deploy/redis_data/' \
  ./ root@107.175.50.44:/opt/sub2api/

ssh -i ~/.ssh/id_ed25519 root@107.175.50.44 '
  cd /opt/sub2api &&
  docker compose -f deploy/docker-compose.local.yml build sub2api &&
  docker compose -f deploy/docker-compose.local.yml up -d sub2api
'
