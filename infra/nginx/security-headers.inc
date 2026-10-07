# Included by every block of rogue.conf that sets headers: a block with its
# own add_header drops the ones it would otherwise inherit. Not a .conf file,
# so nginx does not load it on its own from conf.d.
add_header Strict-Transport-Security "max-age=31536000" always;
add_header X-Content-Type-Options nosniff always;
