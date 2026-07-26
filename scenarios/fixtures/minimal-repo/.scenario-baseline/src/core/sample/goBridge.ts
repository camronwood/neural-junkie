import { invoke } from '@tauri-apps/api';

export async function HelloWorld() {
  await invoke('hello_world');
}