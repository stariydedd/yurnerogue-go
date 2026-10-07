from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.database import Base, engine
from app.models import Run
from app.routers import leaderboard
from app.verifier import rules_version


@asynccontextmanager
async def lifespan(app: FastAPI):
    rules_version()  # Missing or broken verifier must prevent an unsafe deploy.
    # Creates missing tables only (including run_submissions). Changes to
    # existing columns require a migration; create_all does not alter them.
    Base.metadata.create_all(bind=engine)
    # create_all leaves existing tables alone, new indexes included.
    for index in Run.__table__.indexes:
        index.create(bind=engine, checkfirst=True)
    yield


app = FastAPI(title="Rogue 2.0 Leaderboard API", lifespan=lifespan)

# No CORS: the browser game always calls /api on its own origin, and the
# native build is not a browser. Other sites cannot post to the API from a
# visitor's browser.

app.include_router(leaderboard.router)


@app.get("/api/health")
def health():
    """Liveness-проба для деплоя и мониторинга."""
    return {"status": "ok"}
