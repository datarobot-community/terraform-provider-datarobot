from fastapi import FastAPI

app = FastAPI(title="Example C2W Workload")


@app.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/version")
def version() -> str:
    return "1.0.0"
