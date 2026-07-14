||The infinite list of all prime numbers, by the sieve of Eratosthenes.
||
|| Ported to Miracula: primes is still an infinite list; main takes a
|| finite prefix (in the REPL you can also say e.g. take 100 primes).

sieve (p:x) = p : sieve [n | n <- x; n mod p ~= 0]
primes = sieve [2..]

main = show (take 25 primes) ++ "\n"
