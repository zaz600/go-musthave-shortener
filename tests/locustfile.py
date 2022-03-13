import uuid

from locust import HttpUser, task


class ShortenerUser(HttpUser):

    @task
    def shorten(self):
        resp = self.client.post("/api/shorten", json={"url": f"https://ya.ru/{uuid.uuid4()}"})
        # print(resp.json())
        self.client.get(resp.json()["result"], name="/[short_id]")

    @task
    def shorten_batch(self):
        data = [{"original_url": f"https://ya.ru/?{uuid.uuid4()}", "correlation_id": f"{uuid.uuid4()}"} for i in
                range(1000)]
        self.client.post("/api/shorten/batch", json=data)
