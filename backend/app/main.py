from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.database import Base, engine
from app.models import Run
from app.routers import leaderboard
from app.verifier import rules_version


@asynccontextmanager
async def lifespan(app: FastAPI):
    rules_version()  # Отсутствующий или сломанный верификатор должен сорвать небезопасный деплой.
    # Создаёт только недостающие таблицы (включая run_submissions). Изменения
    # существующих столбцов требуют миграции: create_all их не меняет.
    Base.metadata.create_all(bind=engine)
    # create_all не трогает существующие таблицы, в том числе новые индексы к ним.
    for index in Run.__table__.indexes:
        index.create(bind=engine, checkfirst=True)
    yield


app = FastAPI(title="Rogue 2.0 Leaderboard API", lifespan=lifespan)

# Без CORS: браузерная игра всегда обращается к /api со своего адреса, а
# нативная сборка не браузер. Чужие сайты не могут отправлять запросы к API
# из браузера посетителя.

app.include_router(leaderboard.router)


@app.get("/api/health")
def health():
    """Liveness-проба для деплоя и мониторинга."""
    return {"status": "ok"}
