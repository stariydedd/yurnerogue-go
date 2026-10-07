# Подключается в каждом блоке rogue.conf, задающем заголовки: блок со своим
# add_header теряет унаследованные заголовки. Это не файл .conf,
# поэтому nginx сам не загружает его из conf.d.
add_header Strict-Transport-Security "max-age=31536000" always;
add_header X-Content-Type-Options nosniff always;
