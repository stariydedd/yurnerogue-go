import os

DATABASE_URL = os.environ.get(
    "DATABASE_URL",
    "postgresql+psycopg://rogue:rogue@localhost:5432/rogue",
)
