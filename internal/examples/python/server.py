import sys
from struct import pack, unpack
from typing import Callable, BinaryIO, Dict, Tuple
import cbor2


def log(*a):
    print(*a, file=sys.stderr)


class Server(object):
    _methods = {}

    def __new__(cls, *args, **kwargs):
        obj = super(Server, cls).__new__(cls, *args, **kwargs)
        obj.__dict__ = cls._methods
        return obj

    def register(self, name: str, method: Callable[[Dict], Tuple[Dict, str]]):
        self._methods[name] = method

    def call(self, name: str, args: Dict) -> Tuple[Dict, str]:
        return self._methods[name].__call__(args)


class Request:
    def __init__(self):
        self.header = {}
        self.args = {}

    @staticmethod
    def read_or_raise(io: BinaryIO, n: int):
        buf = io.read(n)
        if len(buf) == 0:
            raise EOFError
        return buf

    @staticmethod
    def load(io: BinaryIO):
        self = Request()
        size = unpack("<I", Request.read_or_raise(io, 4))[0]
        self.header = cbor2.loads(Request.read_or_raise(io, size))
        size = unpack("<I", Request.read_or_raise(io, 4))[0]
        self.args = cbor2.loads(Request.read_or_raise(io, size))
        return self

    def service_method(self):
        return self.header["ServiceMethod"]

    def seq(self):
        return self.header["Seq"]

    def __repr__(self):
        from pprint import pformat
        return pformat(vars(self))


class Response(Request):
    def __init__(self, request: Request, reply: any, error: str):
        super().__init__()
        self.header = request.header
        self.args = request.args
        self.reply = reply
        self.header["Error"] = error

    def error(self):
        return self.header["Error"]

    def dumps(self, io: BinaryIO):
        header = cbor2.dumps(self.header)
        reply = cbor2.dumps(self.reply)
        io.write(pack("<I", len(header)))
        io.write(header)
        io.write(pack("<I", len(reply)))
        io.write(reply)


def multiply(args: Dict) -> Tuple[Dict, str]:
    return args["A"] * args["B"], ""


server = Server()
server.register("Arith.Multiply", multiply)

if __name__ == '__main__':
    while not sys.stdin.buffer.closed:
        try:
            req = Request.load(sys.stdin.buffer)
            (result, error) = server.call(req.service_method(), req.args)
            resp = Response(req, result, error)
            resp.dumps(sys.stdout.buffer)
            sys.stdout.buffer.flush()
            log(f"Dispatched {req.service_method()} call")
        except EOFError:
            log("Pipeline closed, exiting")
            break
        except Exception:
            raise
