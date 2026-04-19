import { css } from 'lit';
import tokens from './design-tokens.json';

export const designSystemStyles = css`
  :host {
    --color-primary: ${css`${tokens.color.primary.value}`};
    --color-secondary: ${css`${tokens.color.secondary.value}`};
    --color-danger: ${css`${tokens.color.danger.value}`};
    --color-background: ${css`${tokens.color.background.value}`};
    --color-text: ${css`${tokens.color.text.value}`};
    --font-main: ${css`${tokens.typography.fontFamily.main}`};
    --font-mono: ${css`${tokens.typography.fontFamily.mono}`};
    --font-size-small: ${css`${tokens.typography.fontSize.small}`};
    --font-size-medium: ${css`${tokens.typography.fontSize.medium}`};
    --font-size-large: ${css`${tokens.typography.fontSize.large}`};
    --font-weight-bold: ${css`${tokens.typography.fontWeight.bold}`};
    --font-weight-regular: ${css`${tokens.typography.fontWeight.regular}`};
    --spacing-1: ${css`${tokens.spacing.scale['1']}`};
    --spacing-2: ${css`${tokens.spacing.scale['2']}`};
    --spacing-3: ${css`${tokens.spacing.scale['3']}`};
    --spacing-4: ${css`${tokens.spacing.scale['4']}`};
  }
`;
