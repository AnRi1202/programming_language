long add(long a, long b) {
  return a + b;
}

long mul(long a, long b) {
  return a * b;
}

long f(long x, long y) {
  long (*func_ptr)(long, long);
  
  if (x > 0) {
    func_ptr = add;
  } else {
    func_ptr = mul;
  }
  
  return func_ptr(x, y);
} 