"""Run against the disposable TLS proxy in compose.yml, not production."""
import json
import ssl
import time
import unittest
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen


class ProxyTest(unittest.TestCase):
    def request(self, path, data=None, headers=None):
        request = Request("https://127.0.0.1:8443" + path, data=data, headers=headers or {})
        try:
            response = urlopen(request, context=ssl._create_unverified_context(), timeout=3)
        except HTTPError as error:
            response = error
        with response:
            return response.code, response.headers, response.read()

    def test_submission_limits(self):
        for attempt in range(30):
            try:
                if self.request("/api/health")[0] == 200:
                    break
            except (URLError, TimeoutError, ConnectionError):
                pass
            time.sleep(1)
        else:
            self.fail("test proxy did not become ready")

        self.assertEqual(self.request("/api/runs", b"x" * 65537)[0], 413)
        self.assertEqual(self.request("/api/runs/start", b"{}")[0], 201)
        responses = [self.request("/api/runs", b"{}") for _ in range(16)]
        self.assertEqual(responses[0][0], 201)
        limited = [response for response in responses if response[0] == 429]
        self.assertTrue(limited, responses)
        self.assertEqual(limited[0][1]["Retry-After"], "6")
        self.assertIn("detail", json.loads(limited[0][2]))
        # Changing forwarded headers, query strings or the trailing slash
        # must not reset the key. Nginx uses the actual socket peer address.
        self.assertEqual(self.request("/api/runs/?retry=1", b"{}", {
            "X-Forwarded-For": "203.0.113.1", "X-Real-IP": "203.0.113.2",
        })[0], 429)
        self.assertEqual(self.request("/api/leaderboard")[0], 200)
        self.assertEqual(self.request("/api/runs/start", b"{}")[0], 429)
        self.assertEqual(self.request("/api/health")[0], 200)
        self.assertEqual(self.request("/index.html")[0], 200)


if __name__ == "__main__":
    unittest.main()
