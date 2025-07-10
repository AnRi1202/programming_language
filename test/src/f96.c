typedef long Integer;
typedef double Real;
typedef Integer* IntPtr;
typedef Real* RealPtr;

struct Mixed {
  Integer i;
  Real r;
  IntPtr ip;
  RealPtr rp;
};

Real f(Integer x) {
  Real y = (Real)x / 2.0;
  IntPtr ptr1 = &x;
  RealPtr ptr2 = &y;
  
  struct Mixed m;
  m.i = x;
  m.r = y;
  m.ip = ptr1;
  m.rp = ptr2;
  
  return m.r + (Real)(*m.ip) + *m.rp;
} 