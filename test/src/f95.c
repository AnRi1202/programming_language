long inc(long x) {
  return x + 1;
}

long double_val(long x) {
  return x * 2;
}

long square(long x) {
  return x * x;
}

long f(long x) {
  long result;
  
  // 小さな関数を連続して呼び出し
  result = inc(x);
  result = double_val(result);
  result = square(result);
  result = inc(result);
  result = double_val(result);
  
  return result;
} 