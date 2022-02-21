import logging
import requests

LOGGER = logging.getLogger()
LOGGER.setLevel(logging.INFO)


def handler(event, _):
    LOGGER.info('Event: %s', event)

    response = requests.get("http://example.com")
    LOGGER.info("Got %d status code and %s content-type", response.status_code, response.headers['content-type'])
