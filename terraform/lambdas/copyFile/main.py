import logging

LOGGER = logging.getLogger()
LOGGER.setLevel(logging.INFO)


def handler(event, _):
    LOGGER.info('Event: %s', event)
