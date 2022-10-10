import os

from locust import HttpUser, task


class ListCentrals(HttpUser):
    @task
    def get_centrals(self):
        token = os.getenv('STATIC_TOKEN')
        self.client.get("/api/rhacs/v1/centrals", headers={"authorization": "Bearer " + token})

    # Another task
    # See details: https://docs.locust.io/en/stable/writing-a-locustfile.html#tasks
    # @task
    # def another_task(self):
    #     pass
