import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { RichTextInput } from './RichTextInput';

afterEach(() => {
  cleanup();
});

describe('RichTextInput command text', () => {
  it('keeps typed slash commands in the composer until sent', async () => {
    const onSend = vi.fn();
    render(<RichTextInput onSend={onSend} />);

    const textbox = screen.getByRole('textbox');
    fireEvent.change(textbox, { target: { value: '/help' } });

    expect(textbox).toHaveValue('/help');
    expect(onSend).not.toHaveBeenCalled();

    fireEvent.keyDown(textbox, { key: 'Enter', code: 'Enter' });

    await waitFor(() => {
      expect(onSend).toHaveBeenCalledWith('/help', undefined);
    });
    expect(textbox).toHaveValue('');
  });

  it('keeps composer text when onSend returns false', async () => {
    const onSend = vi.fn().mockResolvedValue(false);
    render(<RichTextInput onSend={onSend} />);

    const textbox = screen.getByRole('textbox');
    fireEvent.change(textbox, { target: { value: '/collaborate test' } });
    fireEvent.keyDown(textbox, { key: 'Enter', code: 'Enter' });

    await waitFor(() => {
      expect(onSend).toHaveBeenCalled();
    });
    expect(textbox).toHaveValue('/collaborate test');
  });

  it('keeps absolute paths as normal composer text', () => {
    const onSend = vi.fn();
    render(<RichTextInput onSend={onSend} />);

    const textbox = screen.getByRole('textbox');
    fireEvent.change(textbox, { target: { value: '/Users/camronwood/development/sandbox' } });

    expect(textbox).toHaveValue('/Users/camronwood/development/sandbox');
    expect(onSend).not.toHaveBeenCalled();
  });
});
