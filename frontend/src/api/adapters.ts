import { Message } from "@bufbuild/protobuf";

/**
 * Funzione di cast sicuro per bypassare l'incertezza del compilatore 
 * su tipi gRPC generati.
 */
export function fromProto<T extends Message>(msg: any): T {
  return msg as T;
}
