import { Message } from "@bufbuild/protobuf";

export function assertType<T extends Message>(data: any): T {
  return data as T;
}
