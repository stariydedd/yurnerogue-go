"""Запускается против одноразового TLS-прокси из compose.yml, а не против прода."""
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

    def wait_ready(self):
        for attempt in range(30):
            try:
                if self.request("/api/health")[0] == 200:
                    break
            except (URLError, TimeoutError, ConnectionError):
                pass
            time.sleep(1)
        else:
            self.fail("test proxy did not become ready")

    def test_bundle_is_served_precompressed(self):
        self.wait_ready()
        cases = {
            "br, gzip": ("br", b"brotli body\n"),
            "gzip, deflate, br;q=0.9": ("br", b"brotli body\n"),
            "gzip": ("gzip", b"gzip body\n"),
            "": (None, b"plain wasm\n"),
        }
        for accept, (encoding, body) in cases.items():
            with self.subTest(accept=accept):
                code, headers, data = self.request("/main.wasm", headers={"Accept-Encoding": accept})
                self.assertEqual(code, 200)
                self.assertEqual(headers["Content-Encoding"], encoding)
                self.assertEqual(headers["Content-Type"], "application/wasm")
                self.assertEqual(headers["Cache-Control"], "no-cache")
                self.assertIn("Accept-Encoding", headers["Vary"])
                self.assertIn("max-age", headers["Strict-Transport-Security"])
                self.assertEqual(data, body)
        # Бандл без своего .br откатывается на gzip.
        code, headers, data = self.request("/old.wasm", headers={"Accept-Encoding": "br, gzip"})
        self.assertEqual((code, headers["Content-Encoding"], data), (200, "gzip", b"old gzip body\n"))
        self.assertEqual(headers["Content-Type"], "application/wasm")
        # Заголовки, которые есть в каждом ответе, в том числе у бандла.
        for path, accept in [("/index.html", ""), ("/main.wasm", "br"), ("/main.wasm", "gzip"),
                             ("/old.wasm", "br"), ("/api/health", "")]:
            with self.subTest(path=path, accept=accept):
                headers = self.request(path, headers={"Accept-Encoding": accept})[1]
                self.assertEqual(headers["X-Content-Type-Options"], "nosniff")
                self.assertIn("max-age", headers["Strict-Transport-Security"])
                self.assertEqual(headers["Server"], "nginx")
        # Сам файл .br не публичный адрес.
        self.assertEqual(self.request("/main.wasm.br", headers={"Accept-Encoding": "br"})[0], 404)
        self.assertEqual(self.request("/index.html")[0], 200)

    def test_submission_limits(self):
        self.wait_ready()
        self.assertEqual(self.request("/api/runs", b"x" * 65537)[0], 413)
        self.assertEqual(self.request("/api/runs/start", b"{}")[0], 201)
        responses = [self.request("/api/runs", b"{}") for _ in range(16)]
        self.assertEqual(responses[0][0], 201)
        limited = [response for response in responses if response[0] == 429]
        self.assertTrue(limited, responses)
        self.assertEqual(limited[0][1]["Retry-After"], "6")
        self.assertIn("detail", json.loads(limited[0][2]))
        # Смена заголовков проксирования, строки запроса или завершающего слеша
        # не должна сбрасывать ключ. Nginx берёт настоящий адрес собеседника сокета.
        self.assertEqual(self.request("/api/runs/?retry=1", b"{}", {
            "X-Forwarded-For": "203.0.113.1", "X-Real-IP": "203.0.113.2",
        })[0], 429)
        self.assertEqual(self.request("/api/leaderboard")[0], 200)
        self.assertEqual(self.request("/api/runs/start", b"{}")[0], 429)
        self.assertEqual(self.request("/api/health")[0], 200)
        self.assertEqual(self.request("/index.html")[0], 200)


if __name__ == "__main__":
    unittest.main()
