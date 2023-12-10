import type {JsonRpcRequest, JsonRpcResponse} from "./types";

export class RpcError extends Error {
    code: number;
    data?: unknown;

    constructor(message: string, code: number, data?: unknown) {
        super(message);
        this.code = code;
        this.data = data;
        Object.setPrototypeOf(this, RpcError.prototype);
    }
}

export type RpcTransport = (
    req: JsonRpcRequest,
    abortSignal: AbortSignal
) => Promise<JsonRpcResponse>;

type RpcClientOptions =
    | string
    | FetchOptions
    | {
    transport: RpcTransport;
};

type FetchOptions = {
    url: string;
    credentials?: RequestCredentials;
    getHeaders?():
        | Record<string, string>
        | Promise<Record<string, string>>
        | undefined;
};

type Promisify<T> = T extends (params: any) => Promise<any> ? T : T extends (params: any) => infer R ? (params: any) => Promise<R> : T;

type PromisifyMethods<T extends object> = {
    [K in keyof T]: Promisify<T[K]>;
};

export function rpcClient<T extends object>(options: RpcClientOptions) {

    if (typeof options === "string") {
        options = {url: options};
    }

    const transport =
        "transport" in options ? options.transport : fetchTransport(options);

    const sendRequest = async (method: string, params: any, signal: AbortSignal) => {
        const res = await transport(createRequest(method, params), signal);
        if ("result" in res) {
            return res.result;
        } else if ("error" in res) {
            const {code, message, data} = res.error;
            throw new RpcError(message, code, data);
        }
        throw new TypeError("Invalid response");
    };

    const abortControllers = new WeakMap<Promise<any>, AbortController>();

    const target = {
        $abort: (promise: Promise<any>) => {
            const ac = abortControllers.get(promise);
            ac?.abort();
        },
    };

    return new Proxy(target, {
        get(target, prop, receiver) {
            if (typeof prop === "symbol") return;
            if (prop in Object.prototype) return;
            if (prop === "toJSON") return;
            if (Reflect.has(target, prop)) {
                return Reflect.get(target, prop, receiver);
            }
            if (prop.startsWith("$")) return;
            return (params: any) => {
                const ac = new AbortController();
                const promise = sendRequest(prop.toString(), params, ac.signal);
                abortControllers.set(promise, ac);
                promise
                    .finally(() => {
                        // Remove the
                        abortControllers.delete(promise);
                    })
                    .catch(() => {
                    });
                return promise;
            };
        },
    }) as typeof target & PromisifyMethods<T>;
}

export function createRequest(method: string, params: any): JsonRpcRequest {
    return {
        jsonrpc: "2.0",
        id: Date.now(),
        method,
        params: params,
    };
}

export function fetchTransport(options: FetchOptions): RpcTransport {
    return async (req: JsonRpcRequest, signal: AbortSignal): Promise<any> => {
        const headers = options?.getHeaders ? await options.getHeaders() : {};
        const res = await fetch(options.url, {
            method: "POST",
            headers: {
                Accept: "application/json",
                "Content-Type": "application/json",
                ...headers,
            },
            body: JSON.stringify(req),
            credentials: options?.credentials,
            signal,
        });
        if (!res.ok) {
            throw new RpcError(res.statusText, res.status);
        }
        return await res.json();
    };
}
