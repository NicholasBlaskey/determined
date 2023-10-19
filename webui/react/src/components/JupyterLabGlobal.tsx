import { matchesShortcut } from 'determined-ui/InputShortcut';
import { useModal } from 'determined-ui/Modal';
import React, { useEffect } from 'react';

import JupyterLabModalComponent from 'components/JupyterLabModal';
import shortCutSettingsConfig, {
  Settings as ShortcutSettings,
} from 'components/UserSettings.settings';
import { keyEmitter, KeyEvent } from 'hooks/useKeyTracker';
import { useSettings } from 'hooks/useSettings';
import { Workspace } from 'types';

interface Props {
  enabled?: boolean;
  workspace?: Workspace;
}

const JupyterLabGlobal: React.FC<Props> = ({ enabled, workspace }) => {
  const JupyterLabModal = useModal(JupyterLabModalComponent);
  const {
    settings: { jupyterLab: jupyterLabShortcut },
  } = useSettings<ShortcutSettings>(shortCutSettingsConfig);

  useEffect(() => {
    const keyDownListener = (e: KeyboardEvent) => {
      if (matchesShortcut(e, jupyterLabShortcut)) {
        JupyterLabModal.open();
      }
    };

    if (enabled) keyEmitter.on(KeyEvent.KeyDown, keyDownListener);

    return () => {
      keyEmitter.off(KeyEvent.KeyDown, keyDownListener);
    };
  }, [JupyterLabModal, jupyterLabShortcut, enabled]);

  return <JupyterLabModal.Component workspace={workspace} />;
};

export default JupyterLabGlobal;
