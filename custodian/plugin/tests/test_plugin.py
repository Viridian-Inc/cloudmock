import os
import unittest

class TestPlugin(unittest.TestCase):
    def test_register_sets_env(self):
        # Clear env vars
        for key in ["AWS_ENDPOINT_URL", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"]:
            os.environ.pop(key, None)

        from c7n_cloudmock.plugin import register
        register()

        self.assertEqual(os.environ["AWS_ENDPOINT_URL"], "http://localhost:4566")
        self.assertEqual(os.environ["AWS_ACCESS_KEY_ID"], "test")

    def test_custom_endpoint(self):
        os.environ["CLOUDMOCK_ENDPOINT"] = "http://custom:9999"
        os.environ.pop("AWS_ENDPOINT_URL", None)

        from c7n_cloudmock.plugin import register
        register()

        self.assertEqual(os.environ["AWS_ENDPOINT_URL"], "http://custom:9999")
