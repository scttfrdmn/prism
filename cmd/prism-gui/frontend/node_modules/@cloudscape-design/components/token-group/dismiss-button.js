// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
import React, { forwardRef } from 'react';
import { getAnalyticsMetadataAttribute } from '@cloudscape-design/component-toolkit/internal/analytics-metadata';
import InternalIcon from '../icon/internal';
import styles from './styles.css.js';
export default forwardRef(DismissButton);
function DismissButton({ disabled, dismissLabel, onDismiss, readOnly }, ref) {
    const analyticsMetadata = {
        action: 'dismiss',
        detail: {
            label: { root: 'self' },
        },
    };
    return (React.createElement("button", Object.assign({ ref: ref, type: "button", className: styles['dismiss-button'], "aria-disabled": disabled || readOnly ? true : undefined, onClick: () => {
            if (disabled || readOnly || !onDismiss) {
                return;
            }
            onDismiss();
        }, "aria-label": dismissLabel }, (disabled || readOnly ? {} : getAnalyticsMetadataAttribute(analyticsMetadata))),
        React.createElement(InternalIcon, { name: "close" })));
}
//# sourceMappingURL=dismiss-button.js.map