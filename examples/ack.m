||defines ackermann's function, beloved of recursion theorists.  Example
||	ack 3 3
||should  yield 61, after doing a huge amount of recursion.  Can only be
||called for small arguments, because the values get so big.
||
|| Ported to Miracula: n+k patterns replaced by guards; the error
|| equation dropped (pattern exhaustion reports a runtime error).

ack m n = n + 1, if m == 0
        = ack (m - 1) 1, if n == 0
        = ack (m - 1) (ack m (n - 1)), otherwise

main = "ack 3 3 = " ++ show (ack 3 3) ++ "\n"
