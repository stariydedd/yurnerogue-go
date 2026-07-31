from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.database import Base, engine
from app.routers import leaderboard


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Пока схема создаётся напрямую; при первом изменении схемы заменим на Alembic-миграции.
    Base.metadata.create_all(bind=engine)
    yield


app = FastAPI(title="Rogue 2.0 Leaderboard API", lifespan=lifespan)

# Игра ходит в API из браузера; на проде nginx отдаёт игру и API с одного домена,
# но для локальной разработки (pygbag-сервер на другом порту) нужен CORS.
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["GET", "POST"],
    allow_headers=["*"],
)

app.include_router(leaderboard.router)


@app.get("/api/health")
def health():
    """Liveness-проба для деплоя и мониторинга."""
    return {"status": "ok"}
