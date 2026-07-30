export interface EditorStateSnapshot {
  uri: string;
  version: number;
}

export function isCurrentEditorState(snapshot: EditorStateSnapshot, current: EditorStateSnapshot): boolean {
  return snapshot.uri === current.uri && snapshot.version === current.version;
}
