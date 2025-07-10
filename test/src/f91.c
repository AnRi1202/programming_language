double f(double x) {
  double arr[4];
  double sum;
  long i;
  
  arr[0] = x;
  arr[1] = x * 1.5;
  arr[2] = x * 2.0;
  arr[3] = x * 2.5;
  
  sum = 0.0;
  for (i = 0; i < 4; i = i + 1) {
    sum = sum + arr[i];
  }
  
  return sum;
} 