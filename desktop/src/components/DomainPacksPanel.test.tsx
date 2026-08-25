import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { DomainPacksPanel } from './DomainPacksPanel';

const setLayoutOwnerMock = vi.fn().mockResolvedValue(undefined);

vi.mock('../stores/packsStore', () => ({
  usePacksStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({
      packs: [
        { id: 'ide', title: 'IDE', enabled: true, installed: true },
        { id: 'software-development', title: 'Software development', enabled: true, installed: true },
      ],
      layoutOwner: 'ide',
      setLayoutOwner: setLayoutOwnerMock,
      hasCapability: () => false,
      applyPacksResponse: vi.fn(),
      fetchPacks: vi.fn(),
    }),
  PACK_CAP: {},
}));

vi.mock('../api/chatAPI', () => ({
  ChatAPI: class {},
}));

vi.mock('./pack-store/PackStoreBrowse', () => ({ PackStoreBrowse: () => <div>Pack store</div> }));
vi.mock('./PackUpdatesBanner', () => ({ PackUpdatesBanner: () => null }));
vi.mock('./pack-store/dev/PackDevStudio', () => ({ PackDevStudio: () => null }));
vi.mock('./MusicCreationToolsPanel', () => ({ MusicCreationToolsPanel: () => null }));
vi.mock('./ModelArenaToolsPanel', () => ({ ModelArenaToolsPanel: () => null }));
vi.mock('./ImageGenerationToolsPanel', () => ({ ImageGenerationToolsPanel: () => null }));

describe('DomainPacksPanel layout owner', () => {
  beforeEach(() => {
    setLayoutOwnerMock.mockClear();
  });

  it('calls setLayoutOwner when layout owner select changes', async () => {
    render(<DomainPacksPanel hubHttp="http://127.0.0.1:9" isActive section="store" />);
    const select = screen.getByLabelText(/UI layout owner/i);
    fireEvent.change(select, { target: { value: 'software-development' } });
    await waitFor(() => {
      expect(setLayoutOwnerMock).toHaveBeenCalledWith('software-development');
    });
  });
});
