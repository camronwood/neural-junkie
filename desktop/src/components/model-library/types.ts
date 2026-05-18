export type StoreModelStatus = 'available' | 'installed' | 'on_disk' | 'cloud';

export interface StoreDetailRow {
  label: string;
  value: string;
}

export interface StoreModelAction {
  id: string;
  label: string;
  variant?: 'primary' | 'secondary' | 'danger';
  disabled?: boolean;
  busyLabel?: string;
  onClick: () => void;
}

export interface StoreModelItem {
  id: string;
  title: string;
  subtitle: string;
  description: string;
  tags: string[];
  sizeHint?: string;
  publisher?: string;
  iconKey?: string;
  status: StoreModelStatus;
  /** Display label for status pill; derived from status if omitted */
  statusLabel?: string;
  externalUrl?: string;
  detailRows?: StoreDetailRow[];
  primaryAction?: StoreModelAction;
  detailActions?: StoreModelAction[];
}

export type ModelStoreView = 'grid' | 'detail';
