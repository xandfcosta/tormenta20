# Os códigos de saída do `backup-db.sh` que o `deploy-prod.sh` lê. Eles moram aqui, carregados pelos dois, para os
# dois scripts dizerem o mesmo número.
#
# BACKUP_NO_DATABASE: o banco ainda não existe — o primeiro deploy, antes de o servidor subir uma vez. É o único
# "não existe" que o deploy aceita como "sem backup": o `.env` ausente também diz "não existe", e lido pelo texto
# ele fazia o deploy seguir sem backup nenhum.
BACKUP_NO_DATABASE=3
