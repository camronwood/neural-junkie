import { describe, expect, it } from 'vitest';
import { withClientPaletteCommands } from './chatWindowHelpers';

describe('withClientPaletteCommands', () => {
  it('includes Neural Canvas once', () => {
    const commands = withClientPaletteCommands([
      {
        name: '/nj-open-neural-canvas',
        description: 'server duplicate',
        category: 'server',
        arguments: [],
      },
    ]);
    expect(commands.filter((command) => command.name === '/nj-open-neural-canvas')).toHaveLength(1);
  });
});
