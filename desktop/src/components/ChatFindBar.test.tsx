import { describe, expect, it, vi, afterEach } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { ChatFindBar } from './ChatFindBar';

describe('ChatFindBar', () => {
  afterEach(() => cleanup());

  it('calls onClose when Escape is pressed', () => {
    const onClose = vi.fn();
    render(<ChatFindBar query="" onQueryChange={() => {}} onClose={onClose} />);
    const input = screen.getByTestId('chat-find-bar').querySelector('input')!;
    fireEvent.keyDown(input, { key: 'Escape' });
    expect(onClose).toHaveBeenCalled();
  });

  it('forwards query changes', () => {
    const onQueryChange = vi.fn();
    render(<ChatFindBar query="" onQueryChange={onQueryChange} onClose={() => {}} />);
    const input = screen.getByTestId('chat-find-bar').querySelector('input')!;
    fireEvent.change(input, { target: { value: 'hello' } });
    expect(onQueryChange).toHaveBeenCalledWith('hello');
  });
});
