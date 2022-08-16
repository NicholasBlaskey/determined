import React from 'react';
import { Link } from 'react-router-dom';

import { paths } from 'routes/utils';
import Icon from 'shared/components/Icon/Icon';
import { getStateColorCssVar } from 'themes';
import { ExperimentBase, TrialDetails } from 'types';

import css from './TrialHeaderLeft.module.scss';

interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

const TrialHeaderLeft: React.FC<Props> = ({ experiment, trial }: Props) => {
  return (
    <div className={css.base}>
      <Link className={css.experiment} to={paths.experimentDetails(trial.experimentId)}>
        Experiment {trial.experimentId} | {experiment.name}
      </Link>
      <Icon name="arrow-right" size="tiny" />
      <div className={css.trial}>
        <div
          className={css.state}
          style={{
            backgroundColor: getStateColorCssVar(trial.state),
            color: getStateColorCssVar(trial.state, { isOn: true, strongWeak: 'strong' }),
          }}>
          {trial.state}
        </div>
        Trial {trial.id}
      </div>
    </div>
  );
};

export default TrialHeaderLeft;
