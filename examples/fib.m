||fib n computes the n'th fibonacci number
||
|| Ported to Miracula: unchanged apart from this main.

fib n = 1,                   if n <= 2
      = fib(n-1) + fib(n-2), otherwise

main = "fib 25 = " ++ show (fib 25) ++ "\n"
