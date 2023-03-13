import logging
import random

import determined as det


def main(core_context, increment_by):
    loss = random.random()
    loss2 = random.random()
    loss3 = random.random()
    loss5 = "a" * 1000000
    for batch in range(1000):
        steps_completed = batch + 1
        loss = loss * (1 - (1 if random.random() > 0.4 else -1) * random.random() * 0.001)
        loss2 = loss2 * (1 - (1 if random.random() > 0.05 else -1) * random.random() * 0.005)
        loss3 = loss3 * (1 - (1 if random.random() > 0.5 else -1) * random.random() * 0.1)
        core_context.train.report_training_metrics(
                steps_completed=steps_completed, metrics={"loss": loss, "loss2": loss2, "loss3": loss3, loss5: 0.837}
        )

    core_context.train.report_validation_metrics(
            steps_completed=steps_completed, metrics={"loss": loss, "loss2": loss2, "loss3": loss3}
    )


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)

    with det.core.init() as core_context:
        main(core_context=core_context, increment_by=1)
