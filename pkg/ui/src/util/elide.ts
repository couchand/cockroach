// TODO(couchand): unify with expandableString component
export function elide(text: string, length: number): string {
  if (text.length <= length + 2) {
    return text;
  }

  return text.substr(0, length).trim() + "…";
}
