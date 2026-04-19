import { css } from 'lit';
import tokens from './design-tokens.json';

export const designSystemStyles = css`
  :host {
    --color-primary: ${css([tokens.color.primary.value] as any)};
    --color-secondary: ${css([tokens.color.secondary.value] as any)};
    --color-danger: ${css([tokens.color.danger.value] as any)};
    --color-background: ${css([tokens.color.background.value] as any)};
    --color-text: ${css([tokens.color.text.value] as any)};
    --font-main: ${css([tokens.typography.fontFamily.main] as any)};
    --font-mono: ${css([tokens.typography.fontFamily.mono] as any)};
    --font-size-small: ${css([tokens.typography.fontSize.small] as any)};
    --font-size-medium: ${css([tokens.typography.fontSize.medium] as any)};
    --font-size-large: ${css([tokens.typography.fontSize.large] as any)};
    --font-weight-bold: ${css([tokens.typography.fontWeight.bold] as any)};
    --font-weight-regular: ${css([tokens.typography.fontWeight.regular] as any)};
    --spacing-1: ${css([tokens.spacing.scale['1']] as any)};
    --spacing-2: ${css([tokens.spacing.scale['2']] as any)};
    --spacing-3: ${css([tokens.spacing.scale['3']] as any)};
    --spacing-4: ${css([tokens.spacing.scale['4']] as any)};
  }
`;
